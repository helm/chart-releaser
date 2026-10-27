// Copyright The Helm Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package github

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v56/github"
)

// ReplaceReleaseAssets replaces a chart archive and its optional provenance in
// an existing mutable release. It returns false when the release does not exist.
// Callers must serialize publishing: GitHub provides no transaction across
// assets, and the subsequent Pages/index update is a separate operation.
func (c *Client) ReplaceReleaseAssets(ctx context.Context, input *Release) (bool, error) {
	// Read immutable explicitly: our go-github version predates this field.
	req, err := c.NewRequest(http.MethodGet, fmt.Sprintf("repos/%s/%s/releases/tags/%s", c.owner, c.repo, url.PathEscape(input.Name)), nil)
	if err != nil {
		return false, err
	}
	var release struct {
		github.RepositoryRelease
		Immutable bool `json:"immutable"`
	}
	resp, err := c.Do(ctx, req, &release)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}
	if release.Immutable {
		return true, fmt.Errorf("release %s is immutable; publish a new chart version instead", input.Name)
	}
	if release.GetID() == 0 {
		return true, errors.New("release response has no ID")
	}
	if len(input.Assets) == 0 || !strings.HasSuffix(input.Assets[0].Path, ".tgz") {
		return true, errors.New("replacement requires a chart archive")
	}
	archive := filepath.Base(input.Assets[0].Path)
	for i, asset := range input.Assets {
		if i > 1 || (i == 1 && filepath.Base(asset.Path) != archive+".prov") {
			return true, errors.New("replacement accepts only one chart archive and its provenance")
		}
		f, err := os.Open(asset.Path)
		if err != nil {
			return true, err
		}
		stat, err := f.Stat()
		_ = f.Close()
		if err != nil {
			return true, err
		}
		if !stat.Mode().IsRegular() {
			return true, fmt.Errorf("asset %s is not a regular file", asset.Path)
		}
	}

	var originals []*github.ReleaseAsset
	opts := &github.ListOptions{PerPage: 100}
	for {
		assets, resp, err := c.Repositories.ListReleaseAssets(ctx, c.owner, c.repo, release.GetID(), opts)
		if err != nil {
			return true, err
		}
		for _, asset := range assets {
			if asset.GetName() == archive || asset.GetName() == archive+".prov" {
				originals = append(originals, asset)
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	// Stage uploads under unique names before changing any original asset.
	// Suffixes deliberately do not end in .tgz, so index generation ignores them.
	suffix := ".cr-replace-" + rand.Text()
	var staged []*github.ReleaseAsset
	var renamed []*github.ReleaseAsset
	rollback := func(cause error) error {
		// Cleanup must still be attempted after cancellation of the operation.
		cleanupCtx := context.WithoutCancel(ctx)
		for _, asset := range staged {
			if _, err := c.Repositories.DeleteReleaseAsset(cleanupCtx, c.owner, c.repo, asset.GetID()); err != nil {
				cause = errors.Join(cause, fmt.Errorf("removing staged asset %d: %w", asset.GetID(), err))
			}
		}
		for _, asset := range renamed {
			if _, _, err := c.Repositories.EditReleaseAsset(cleanupCtx, c.owner, c.repo, asset.GetID(), &github.ReleaseAsset{Name: asset.Name}); err != nil {
				cause = errors.Join(cause, fmt.Errorf("restoring asset %d to %s: %w", asset.GetID(), asset.GetName(), err))
			}
		}
		return cause
	}
	for _, asset := range input.Assets {
		f, err := os.Open(asset.Path)
		if err != nil {
			return true, rollback(err)
		}
		uploaded, _, err := c.Repositories.UploadReleaseAsset(ctx, c.owner, c.repo, release.GetID(), &github.UploadOptions{Name: filepath.Base(asset.Path) + suffix}, f)
		_ = f.Close()
		if err != nil {
			return true, rollback(fmt.Errorf("staging replacement for %s: %w", asset.Path, err))
		}
		staged = append(staged, uploaded)
	}
	for _, asset := range originals {
		backupName := asset.GetName() + suffix + "-backup"
		// Include even an ambiguous failed rename in rollback: the server may
		// have applied it before the response was lost.
		renamed = append(renamed, asset)
		if _, _, err := c.Repositories.EditReleaseAsset(ctx, c.owner, c.repo, asset.GetID(), &github.ReleaseAsset{Name: &backupName}); err != nil {
			return true, rollback(err)
		}
	}
	for i, asset := range staged {
		name := filepath.Base(input.Assets[i].Path)
		if _, _, err := c.Repositories.EditReleaseAsset(ctx, c.owner, c.repo, asset.GetID(), &github.ReleaseAsset{Name: &name}); err != nil {
			return true, rollback(err)
		}
	}
	for _, asset := range originals {
		if _, err := c.Repositories.DeleteReleaseAsset(ctx, c.owner, c.repo, asset.GetID()); err != nil {
			// New assets are already in place. Continue to allow the caller to
			// publish their index; report the retained backup for manual cleanup.
			fmt.Fprintf(os.Stderr, "WARNING: replacement succeeded but backup asset %d (%s) could not be removed: %v\n", asset.GetID(), asset.GetName()+suffix+"-backup", err)
		}
	}
	return true, nil
}
