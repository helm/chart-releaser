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

package releaser

import (
	"testing"

	"github.com/helm/chart-releaser/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
)

func TestParseChangelogSectionMarkdown(t *testing.T) {
	for _, tt := range []struct{ name, section string }{
		{"backtick fence", "## [1.2.3]\n\n```sh\n# run this first\necho hello\n```\n\n- important migration"},
		{"tilde fence", "## [1.2.3]\n\n~~~sh\n## [1.2.2]\n~~~\n\n- important migration"},
		{"shorter fence does not close", "## [1.2.3]\n\n````sh\n```\n# comment\n````\n\n- notes"},
		{"different fence does not close", "## [1.2.3]\n\n```sh\n~~~\n# comment\n```\n\n- notes"},
		{"fence with text does not close", "## [1.2.3]\n\n```sh\n``` trailing text\n# comment\n```\n\n- notes"},
		{"indented fence", "## [1.2.3]\n\n   ```sh\n# comment\n   ```\n\n- notes"},
		{"inline version link", "## [1.2.3](https://github.com/example/chart/compare/v1.2.2...v1.2.3) (2024-01-02)\n\n- notes"},
		{"parenthesized date", "## 1.2.3 (2024-01-02)\n\n- notes"},
		{"leading spaces and closing hashes", "   ## [1.2.3] - 2024-01-02 ##\n\n- notes"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := "# Changelog\n\n```md\n## [1.2.3]\nnot a real release\n```\n\n" + tt.section + "\n\n## [1.2.2]\nold"
			assert.Equal(t, tt.section, parseChangelogSection(input, "1.2.3"))
		})
	}
	t.Run("CRLF", func(t *testing.T) {
		assert.Equal(t, "## [1.2.3]\n\n- notes", parseChangelogSection("## [1.2.3]\r\n\r\n- notes\r\n", "1.2.3"))
	})
	t.Run("version prefix is not a match", func(t *testing.T) {
		assert.Empty(t, parseChangelogSection("## [1.2.30]\n- wrong", "1.2.3"))
	})
}

func TestReleaser_ChangelogWithGeneratedReleaseNotes(t *testing.T) {
	packagePath := t.TempDir()
	_, err := chartutil.Save(&chart.Chart{
		Metadata: &chart.Metadata{APIVersion: "v2", Name: "test-chart", Version: "1.2.3", Description: "fallback"},
		Files:    []*chart.File{{Name: "CHANGELOG.md", Data: []byte(testKeepAChangelog)}},
	}, packagePath)
	require.NoError(t, err)
	fakeGitHub := new(FakeGitHub)
	fakeGitHub.On("CreateRelease", mock.Anything, mock.Anything).Return(nil)
	fakeGitHub.On("ListTagsWithPrefix", mock.Anything, "test-chart-").Return([]string{"test-chart-1.2.2"}, nil)
	fakeGitHub.On("GenerateReleaseNotes", mock.Anything, "test-chart-1.2.3", "test-chart-1.2.2", "").Return("## What's Changed\n- generated", nil)
	r := &Releaser{
		config: &config.Options{PackagePath: packagePath, ReleaseNameTemplate: "{{ .Name }}-{{ .Version }}", ReleaseNotesSource: "changelog", GenerateReleaseNotes: true},
		github: fakeGitHub,
	}
	require.NoError(t, r.CreateReleases())
	assert.Equal(t, "## [1.2.3] - 2024-01-02\n\n### Added\n\n- matching version notes\n\n## What's Changed\n- generated", fakeGitHub.release.Description)
	assert.False(t, fakeGitHub.release.GenerateReleaseNotes)
	fakeGitHub.AssertExpectations(t)
}
