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

package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Git struct{}

// AddWorktree creates a new Git worktree with a detached HEAD for the given committish and returns its path.
func (g *Git) AddWorktree(workingDir string, committish string) (string, error) {
	dir, err := os.MkdirTemp("", "chart-releaser-")
	if err != nil {
		return "", err
	}
	command := exec.Command("git", "worktree", "add", "--detach", dir, committish)

	if err := runCommand(workingDir, command); err != nil {
		return "", err
	}
	return dir, nil
}

// RemoveWorktree removes the Git worktree with the given path.
func (g *Git) RemoveWorktree(workingDir string, path string) error {
	command := exec.Command("git", "worktree", "remove", path, "--force")
	return runCommand(workingDir, command)
}

// Add runs 'git add' with the given args.
func (g *Git) Add(workingDir string, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("no args specified")
	}
	addArgs := []string{"add"}
	addArgs = append(addArgs, args...)
	command := exec.Command("git", addArgs...)
	return runCommand(workingDir, command)
}

// Commit runs 'git commit' with the given message. the commit is signed off.
func (g *Git) Commit(workingDir string, message string) error {
	command := exec.Command("git", "commit", "--message", message, "--signoff")
	return runCommand(workingDir, command)
}

// UpdateBranch runs 'git pull' with the given args.
func (g *Git) Pull(workingDir string, args ...string) error {
	pullArgs := []string{"pull"}
	pullArgs = append(pullArgs, args...)
	command := exec.Command("git", pullArgs...)
	return runCommand(workingDir, command)
}

// Push runs 'git push' with the given args.
func (g *Git) Push(workingDir string, args ...string) error {
	pushArgs := []string{"push"}
	pushArgs = append(pushArgs, args...)
	command := exec.Command("git", pushArgs...)
	return runCommand(workingDir, command)
}

// GetPushURL returns the push url with a token inserted
func (g *Git) GetPushURL(remote string, token string) (string, error) {
	pushURL, err := exec.Command("git", "remote", "get-url", "--push", remote).Output()
	if err != nil {
		return "", err
	}

	pushURLStr := string(pushURL)
	found := false

	if pushURLStr, found = strings.CutPrefix(pushURLStr, "git@"); found {
		pushURLStr = strings.ReplaceAll(pushURLStr, ":", "/")
		pushURLStr = strings.TrimSuffix(pushURLStr, ".git\n")
	} else {
		pushURLStr = strings.TrimPrefix(pushURLStr, "https://")
	}
	pushURLWithToken := fmt.Sprintf("https://x-access-token:%s@%s", token, strings.Trim(pushURLStr, "\n"))
	return pushURLWithToken, nil
}

func runCommand(workingDir string, command *exec.Cmd) error {
	command.Dir = workingDir
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}
