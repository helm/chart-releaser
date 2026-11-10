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
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gh "github.com/google/go-github/v56/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type contextTransport func(*http.Request) (*http.Response, error)

func (f contextTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClientHonorsRequestContext(t *testing.T) {
	create := func(c *Client, ctx context.Context, asset string) error {
		return c.CreateRelease(ctx, &Release{Name: "chart-1.0.0", Assets: []*Asset{{Path: asset}}})
	}
	for _, tc := range []struct {
		name            string
		call            func(*Client, context.Context, string) error
		cancelAfter     string
		failAfterCancel bool
		wantRequests    []string
	}{
		{name: "get release", call: func(c *Client, ctx context.Context, _ string) error { _, err := c.GetRelease(ctx, "v1"); return err }},
		{name: "generate notes", call: func(c *Client, ctx context.Context, _ string) error {
			_, err := c.GenerateReleaseNotes(ctx, "v1", "", "")
			return err
		}},
		{name: "list tags", call: func(c *Client, ctx context.Context, _ string) error {
			_, err := c.ListTagsWithPrefix(ctx, "chart-")
			return err
		}},
		{name: "create release", call: create},
		{name: "create with notes", call: func(c *Client, ctx context.Context, _ string) error {
			return c.CreateRelease(ctx, &Release{Name: "v1", GenerateReleaseNotes: true})
		}},
		{name: "upload asset", call: func(c *Client, ctx context.Context, asset string) error { return c.uploadReleaseAsset(ctx, 42, asset) }},
		{name: "cancel after draft creation", call: create, cancelAfter: "POST /repos/owner/repo/releases", wantRequests: []string{"POST /repos/owner/repo/releases"}},
		{name: "cancel during upload", call: create, cancelAfter: "POST /repos/owner/repo/releases/42/assets", failAfterCancel: true, wantRequests: []string{"POST /repos/owner/repo/releases", "POST /repos/owner/repo/releases/42/assets"}},
		{name: "cancel between tag pages", call: func(c *Client, ctx context.Context, _ string) error {
			_, err := c.ListTagsWithPrefix(ctx, "chart-")
			return err
		}, cancelAfter: "GET /repos/owner/repo/git/matching-refs/tags/chart-", wantRequests: []string{"GET /repos/owner/repo/git/matching-refs/tags/chart-"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancelAfter == "" {
				cancel()
			}
			var requests []string
			transport := contextTransport(func(req *http.Request) (*http.Response, error) {
				if err := req.Context().Err(); err != nil {
					return nil, err
				}
				request := req.Method + " " + req.URL.Path
				requests = append(requests, request)
				if request == tc.cancelAfter {
					cancel()
					if tc.failAfterCancel {
						return nil, context.Canceled
					}
				}
				header := http.Header{"Content-Type": {"application/json"}}
				body := `{"id":42,"body":"notes"}`
				if strings.Contains(req.URL.Path, "/git/matching-refs/") {
					body = `[]`
					if req.URL.Query().Get("page") == "" {
						next := *req.URL
						query := next.Query()
						query.Set("page", "2")
						next.RawQuery = query.Encode()
						header.Set("Link", "<"+next.String()+">; rel=\"next\"")
					}
				}
				return &http.Response{StatusCode: http.StatusOK, Header: header, Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			client := &Client{owner: "owner", repo: "repo", Client: gh.NewClient(&http.Client{Transport: transport})}
			asset := filepath.Join(t.TempDir(), "chart.tgz")
			require.NoError(t, os.WriteFile(asset, []byte("chart"), 0o600))
			err := tc.call(client, ctx, asset)
			require.ErrorIs(t, err, context.Canceled)
			assert.Equal(t, tc.wantRequests, requests)
		})
	}
}
