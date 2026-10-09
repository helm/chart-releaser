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
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type Git struct{}

// AddWorktree creates a new Git worktree with a detached HEAD for the given commit-ish and returns its path.
func (g *Git) AddWorktree(workingDir string, commitIsh string) (string, error) {
	dir, err := os.MkdirTemp("", "chart-releaser-")
	if err != nil {
		return "", err
	}
	command := exec.Command("git", "worktree", "add", "--detach", dir, commitIsh)

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
	addArgs := make([]string, 0, 1+len(args))
	addArgs = append(addArgs, "add")
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
	pullArgs := make([]string, 0, 1+len(args))
	pullArgs = append(pullArgs, "pull")
	pullArgs = append(pullArgs, args...)
	command := exec.Command("git", pullArgs...)
	return runCommand(workingDir, command)
}

// Push runs 'git push' with the given args.
func (g *Git) Push(workingDir string, args ...string) error {
	pushArgs := make([]string, 0, 1+len(args))
	pushArgs = append(pushArgs, "push")
	pushArgs = append(pushArgs, args...)
	command := exec.Command("git", pushArgs...)
	return runCommand(workingDir, command)
}

// GetPushURL returns the push url with a token inserted. The remote can use any
// of the URL forms git understands (https, http, ssh:// or scp-like
// git@host:owner/repo.git); the token is always sent over HTTP(S).
func (g *Git) GetPushURL(remote string, token string) (string, error) {
	pushURL, err := exec.Command("git", "remote", "get-url", "--push", remote).Output()
	if err != nil {
		return "", err
	}

	return buildPushURL(strings.TrimSpace(string(pushURL)), token)
}

func buildPushURL(remoteURL string, token string) (string, error) {
	scheme, host, path := "https", "", ""

	if strings.Contains(remoteURL, "://") {
		u, err := url.Parse(remoteURL)
		if err != nil {
			return "", fmt.Errorf("unable to parse git remote URL: %w", err)
		}
		if u.Scheme == "http" {
			scheme = "http"
		}
		host, path = u.Host, u.Path
		if u.Scheme != "https" && u.Scheme != "http" {
			// ssh:// and git:// remotes may use a port that is not valid for HTTPS
			host = u.Hostname()
		}
	} else {
		// scp-like syntax: [user@]host:path
		hostAndUser, repoPath, found := strings.Cut(remoteURL, ":")
		if !found {
			return "", fmt.Errorf("unsupported git remote URL %q", remoteURL)
		}
		host = hostAndUser[strings.LastIndex(hostAndUser, "@")+1:]
		path = repoPath
	}

	path = strings.TrimPrefix(path, "/")
	if host == "" || path == "" {
		return "", fmt.Errorf("unsupported git remote URL %q", remoteURL)
	}

	return fmt.Sprintf("%s://x-access-token:%s@%s/%s", scheme, token, host, path), nil
}

func runCommand(workingDir string, command *exec.Cmd) error {
	command.Dir = workingDir
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}
