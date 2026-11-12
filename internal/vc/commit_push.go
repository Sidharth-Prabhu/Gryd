package vc

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// HasUncommittedChanges checks if there are modified, staged, or untracked files.
func HasUncommittedChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}

// StageAll stages all changes in the repository.
func StageAll() error {
	cmd := exec.Command("git", "add", "-A")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// Commit performs a git commit with the given message.
func Commit(message string) (string, error) {
	if strings.TrimSpace(message) == "" {
		return "", fmt.Errorf("commit message cannot be empty")
	}

	cmd := exec.Command("git", "commit", "-m", message)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("commit failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}

	return strings.TrimSpace(string(out)), nil
}

// PromptForCommitMessage asks the user to enter a commit message.
func PromptForCommitMessage(defaultMsg string) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	prompt := "Enter commit message: "
	if defaultMsg != "" {
		prompt = fmt.Sprintf("Enter commit message (default '%s'): ", defaultMsg)
	}

	fmt.Print(prompt)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	msg := strings.TrimSpace(text)
	if msg == "" && defaultMsg != "" {
		return defaultMsg, nil
	}
	if msg == "" {
		return "", fmt.Errorf("commit message cannot be empty")
	}
	return msg, nil
}

// GetCurrentBranch returns the name of the active branch.
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Push pushes the current branch to the specified remote repository.
func Push(remote, branch string) error {
	cmd := exec.Command("git", "push", "-u", remote, branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
