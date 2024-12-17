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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Songmu/retry"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chart"

	"github.com/blang/semver"
	"github.com/google/go-github/v56/github"
	"golang.org/x/oauth2"
)

type Release struct {
	Name        string
	Description string
	Assets      []*Asset
	Commit      string
	MakeLatest  string
	SemVer      semver.Version
}

type Asset struct {
	Path string
	URL  string
}

// Client is the client for interacting with the GitHub API
type Client struct {
	owner string
	repo  string
	*github.Client
}

// NewClient creates and initializes a new GitHubClient
func NewClient(owner, repo, token, baseURL, uploadURL string) *Client {
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

	if baseEndpoint, err := url.Parse(baseURL); err == nil {
		if !strings.HasSuffix(baseEndpoint.Path, "/") {
			baseEndpoint.Path += "/"
		}
		client.BaseURL = baseEndpoint
	}

	if uploadEndpoint, err := url.Parse(uploadURL); err == nil {
		if !strings.HasSuffix(uploadEndpoint.Path, "/") {
			uploadEndpoint.Path += "/"
		}
		client.UploadURL = uploadEndpoint
	}

	return &Client{
		owner:  owner,
		repo:   repo,
		Client: client,
	}
}

// GetRelease queries the GitHub API for a specified release object
func (c *Client) GetRelease(_ context.Context, tag string) (*Release, error) {
	// Check Release whether already exists or not
	release, _, err := c.Repositories.GetReleaseByTag(context.TODO(), c.owner, c.repo, tag)
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

// GetLatestChartRelease queries the GitHub API for the previous release of a chart
func (c *Client) GetLatestChartRelease(_ context.Context, prefix string) (*Release, error) {
	// Append hyphen to prefix unless already present
	prefix = strings.TrimSuffix(prefix, "-") + "-"

	// Find all versions with tags matching prefix
	opt := &github.ListOptions{
		PerPage: 100,
	}
	var versions []semver.Version
	for {
		rels, resp, err := c.Repositories.ListReleases(context.TODO(), c.owner, c.repo, opt)
		if err != nil {
			return nil, err
		} else if len(rels) == 0 {
			return nil, errors.New("no releases found")
		}
		for _, rel := range rels {
			if strings.HasPrefix(*rel.TagName, prefix) {
				version := semver.MustParse(strings.TrimPrefix(*rel.TagName, prefix))
				versions = append(versions, version)
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	// Sort versions ascending
	semver.Sort(versions)

	// Find highest version
	latestVersion := versions[len(versions)-1]
	var release *github.RepositoryRelease
	if rel, _, err := c.Repositories.GetReleaseByTag(context.TODO(), c.owner, c.repo, prefix+latestVersion.String()); err == nil {
		release = rel
	}

	result := &Release{
		Name:   *release.TagName,
		Commit: *release.TargetCommitish,
		SemVer: latestVersion,
	}
	return result, nil
}

// GenerateReleaseNotes generates the release notes for a release
func (c *Client) GenerateReleaseNotes(_ context.Context, latestRelease *Release, chart *chart.Chart) (string, error) {
	notes, _, err := c.Repositories.GenerateReleaseNotes(context.TODO(), c.owner, c.repo, &github.GenerateNotesOptions{
		TagName:         chart.Metadata.Name + "-" + chart.Metadata.Version,
		PreviousTagName: &latestRelease.Name,
	})
	if err != nil {
		return "", err
	}
	return notes.Body, err
}

// CreateRelease creates a new release object in the GitHub API
func (c *Client) CreateRelease(_ context.Context, input *Release) error {
	req := &github.RepositoryRelease{
		Name:            &input.Name,
		Body:            &input.Description,
		TagName:         &input.Name,
		TargetCommitish: &input.Commit,
		MakeLatest:      &input.MakeLatest,
	}

	release, _, err := c.Repositories.CreateRelease(context.TODO(), c.owner, c.repo, req)
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

// CreatePullRequest creates a pull request in the repository specified by repoURL.
// The return value is the pull request URL.
func (c *Client) CreatePullRequest(owner string, repo string, message string, head string, base string) (string, error) {
	split := strings.SplitN(message, "\n", 2)
	title := split[0]

	pr := &github.NewPullRequest{
		Title: &title,
		Head:  &head,
		Base:  &base,
	}
	if len(split) == 2 {
		body := strings.TrimSpace(split[1])
		pr.Body = &body
	}

	pullRequest, _, err := c.PullRequests.Create(context.Background(), owner, repo, pr)
	if err != nil {
		return "", err
	}
	return *pullRequest.HTMLURL, nil
}

// UploadAsset uploads specified assets to a given release object
func (c *Client) uploadReleaseAsset(_ context.Context, releaseID int64, filename string) error {
	filename, err := filepath.Abs(filename)
	if err != nil {
		return errors.Wrap(err, "failed to get abs path")
	}

	opts := &github.UploadOptions{
		// Use base name by default
		Name: filepath.Base(filename),
	}

	if err := retry.Retry(3, 3*time.Second, func() error { //nolint: revive
		f, err := os.Open(filename)
		if err != nil {
			return errors.Wrap(err, "failed to open file")
		}
		defer f.Close()
		if _, _, err = c.Repositories.UploadReleaseAsset(context.TODO(), c.owner, c.repo, releaseID, opts, f); err != nil {
			return errors.Wrapf(err, "failed to upload release asset: %s", filename)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
