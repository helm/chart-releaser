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

package config

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigurationPreRelease(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantLatest bool
		wantPre    bool
		wantErr    bool
	}{
		{name: "defaults", args: nil, wantLatest: true, wantPre: false},
		{name: "pre-release disables latest", args: []string{"--pre-release"}, wantLatest: false, wantPre: true},
		{name: "pre-release with latest explicitly disabled", args: []string{"--pre-release", "--make-release-latest=false"}, wantLatest: false, wantPre: true},
		{name: "pre-release with latest explicitly enabled", args: []string{"--pre-release", "--make-release-latest=true"}, wantErr: true},
		{name: "latest explicitly enabled without pre-release", args: []string{"--make-release-latest=true"}, wantLatest: true, wantPre: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "upload"}
			cmd.Flags().Bool("make-release-latest", true, "")
			cmd.Flags().Bool("pre-release", false, "")
			require.NoError(t, cmd.ParseFlags(tt.args))

			opts, err := LoadConfiguration("", cmd, nil)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantLatest, opts.MakeReleaseLatest)
			assert.Equal(t, tt.wantPre, opts.PreRelease)
		})
	}
}
