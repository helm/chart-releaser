package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

type Git struct{}

// AddWorktree creates a new Git worktree with a detached HEAD for the given committish and returns its path.
func (g *Git) AddWorktree(workingDir string, committish string) (string, error) {
	dir, err := ioutil.TempDir("", "chart-releaser-")
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

// Push runs 'git push' with the given args.
func (g *Git) Push(workingDir string, args ...string) error {
	pushArgs := []string{"push"}
	pushArgs = append(pushArgs, args...)
	command := exec.Command("git", pushArgs...)
	return runCommand(workingDir, command)
}

func runCommand(workingDir string, command *exec.Cmd) error {
	command.Dir = workingDir
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}
