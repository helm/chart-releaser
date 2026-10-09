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
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-github/v56/github"
	"github.com/stretchr/testify/assert"
)

func TestIsErrTagAlreadyExist(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "tag already exists",
			err:  &github.ErrorResponse{Errors: []github.Error{{Resource: "Release", Field: "tag_name", Code: "already_exists"}}},
			want: true,
		},
		{
			name: "wrapped tag already exists",
			err:  fmt.Errorf("wrapped: %w", &github.ErrorResponse{Errors: []github.Error{{Field: "tag_name", Code: "already_exists"}}}),
			want: true,
		},
		{
			name: "other field already exists",
			err:  &github.ErrorResponse{Errors: []github.Error{{Field: "name", Code: "already_exists"}}},
			want: false,
		},
		{
			name: "other code on tag_name",
			err:  &github.ErrorResponse{Errors: []github.Error{{Field: "tag_name", Code: "invalid"}}},
			want: false,
		},
		{
			name: "not a github error",
			err:  errors.New("boom"),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isErrTagAlreadyExist(tt.err))
		})
	}
}
