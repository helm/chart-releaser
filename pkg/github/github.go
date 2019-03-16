// Copyright The Helm Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package github

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/Songmu/retry"
	"github.com/pkg/errors"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// GitHub contains the functions necessary for interacting with GitHub release
// objects
type GitHub interface {
	GetRepository(ctx context.Context) (*github.Repository, error)
	CreateRelease(ctx context.Context, req *github.RepositoryRelease) (*github.RepositoryRelease, error)
	GetRelease(ctx context.Context, tag string) (*Release, error)
	ListReleases(ctx context.Context) ([]*github.RepositoryRelease, error)
	UploadAsset(ctx context.Context, releaseID int64, filename string) error
}

type Release struct {
	Name   string
	Assets []*Asset
}

type Asset struct {
	Path string
	URL  string
}

// Client is the client for interacting with the GitHub API
type Client struct {
	Owner string
	Repo  string
	*github.Client
}

// NewClient creates and initializes a new GitHubClient
func NewClient(owner, repo, token string) *Client {
	var client *github.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: token,
		})
		tc := oauth2.NewClient(context.TODO(), ts)
		client = github.NewClient(tc)
	} else {
		client = github.NewClient(nil)
	}
	return &Client{
		Owner:  owner,
		Repo:   repo,
		Client: client,
	}
}

// GetRelease queries the GitHub API for a specified release object
func (c *Client) GetRelease(ctx context.Context, tag string) (*Release, error) {
	// Check Release whether already exists or not
	release, _, err := c.Repositories.GetReleaseByTag(context.TODO(), c.Owner, c.Repo, tag)
	if err != nil {
		return nil, err
	}

	result := &Release{
		Assets: []*Asset{},
	}
	for _, ass := range release.Assets {
		asset := &Asset{*ass.Name, *ass.BrowserDownloadURL}
		result.Assets = append(result.Assets, asset)
	}
	return result, nil
}

// ListReleases lists Releases given a repository
func (c *Client) ListReleases(ctx context.Context) ([]*Release, error) {
	var result []*Release
	page := 1

	for {
		repoReleases, res, err := c.Repositories.ListReleases(context.TODO(), c.Owner, c.Repo, &github.ListOptions{Page: page})
		if err != nil {
			return nil, errors.Wrap(err, "failed to list releases")
		}

		for _, repoRelease := range repoReleases {
			release := &Release{
				Name:   *repoRelease.Name,
				Assets: []*Asset{},
			}
			for _, ass := range repoRelease.Assets {
				asset := &Asset{*ass.Name, *ass.BrowserDownloadURL}
				release.Assets = append(release.Assets, asset)
			}

			result = append(result, release)
		}

		if res.NextPage <= page {
			break
		}
		page = res.NextPage
	}
	return result, nil
}

// CreateRelease creates a new release object in the GitHub API
func (c *Client) CreateRelease(ctx context.Context, input *Release) error {
	req := &github.RepositoryRelease{
		Name:    &input.Name,
		TagName: &input.Name,
	}

	release, _, err := c.Repositories.CreateRelease(context.TODO(), c.Owner, c.Repo, req)
	if err != nil {
		return err
	}

	for _, asset := range input.Assets {
		if err := c.uploadReleaseAsset(context.TODO(), *release.ID, asset.Path); err != nil {
			return err
		}
	}
	return nil
}

// UploadAsset uploads specified assets to a given release object
func (c *Client) uploadReleaseAsset(ctx context.Context, releaseID int64, filename string) error {

	filename, err := filepath.Abs(filename)
	if err != nil {
		return errors.Wrap(err, "failed to get abs path")
	}

	f, err := os.Open(filename)
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}

	opts := &github.UploadOptions{
		// Use base name by default
		Name: filepath.Base(filename),
	}

	err = retry.Retry(3, 3*time.Second, func() error {
		if _, _, err = c.Repositories.UploadReleaseAsset(context.TODO(), c.Owner, c.Repo, releaseID, opts, f); err != nil {
			return errors.Wrapf(err, "failed to upload release asset: %s\n", filename)
		}
		return nil
	})

	return nil
}
