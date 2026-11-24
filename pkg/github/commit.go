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
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

// CreateSignedCommit updates one file on a branch using GitHub's server-side
// signing. expectedHead must be the revision used to prepare contents.
func (c *Client) CreateSignedCommit(ctx context.Context, branch, expectedHead, path, message string, contents []byte) error {
	headline, body, _ := strings.Cut(message, "\n")
	input := map[string]any{
		"branch":          map[string]string{"repositoryNameWithOwner": c.owner + "/" + c.repo, "branchName": branch},
		"expectedHeadOid": expectedHead,
		"message":         map[string]string{"headline": headline, "body": strings.TrimLeft(body, "\r\n")},
		"fileChanges": map[string]any{"additions": []map[string]string{{
			"path": path, "contents": base64.StdEncoding.EncodeToString(contents),
		}}},
	}
	payload := struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}{
		Query: `mutation($input: CreateCommitOnBranchInput!) {
   createCommitOnBranch(input: $input) { commit { oid } }
  }`,
		Variables: map[string]any{"input": input},
	}

	// GitHub.com uses /graphql; Enterprise uses /api/graphql instead of /api/v3.
	endpoint := *c.BaseURL
	endpoint.Path = strings.TrimSuffix(strings.TrimRight(endpoint.Path, "/"), "/v3") + "/graphql"
	req, err := c.NewRequest(http.MethodPost, endpoint.String(), payload)
	if err != nil {
		return fmt.Errorf("building commit mutation: %w", err)
	}
	var result struct {
		Data struct {
			CreateCommitOnBranch struct {
				Commit struct {
					OID string `json:"oid"`
				} `json:"commit"`
			} `json:"createCommitOnBranch"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if _, err := c.Do(ctx, req, &result); err != nil {
		return fmt.Errorf("commit mutation: %w", err)
	}
	if len(result.Errors) > 0 {
		messages := make([]string, 0, len(result.Errors))
		for _, err := range result.Errors {
			messages = append(messages, err.Message)
		}
		return fmt.Errorf("commit mutation: %s", strings.Join(messages, "; "))
	}
	if result.Data.CreateCommitOnBranch.Commit.OID == "" {
		return fmt.Errorf("commit mutation returned no commit")
	}
	return nil
}
