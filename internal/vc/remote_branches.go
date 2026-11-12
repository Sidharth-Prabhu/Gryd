package vc

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// BranchInfo holds information about a Git branch.
type BranchInfo struct {
	Name      string
	IsCurrent bool
	IsRemote  bool
	Hash      string
	Message   string
}

// RemoteInfo holds information about a Git remote.
type RemoteInfo struct {
	Name string
	URL  string
	Type string // fetch or push
}

// IsGitRepo checks if current directory or ancestors are inside a git repository.
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	return err == nil
}

// GetBranches returns all local and remote branches.
func GetBranches() ([]BranchInfo, error) {
	if !IsGitRepo() {
		return nil, fmt.Errorf("current directory is not a Git repository")
	}

	cmd := exec.Command("git", "branch", "-a", "-vv", "--no-color")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	var branches []BranchInfo
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimRight(line, " \t\r")
		if len(line) == 0 {
			continue
		}

		isCurrent := strings.HasPrefix(line, "*")
		cleanLine := strings.TrimSpace(strings.TrimPrefix(line, "*"))

		parts := strings.Fields(cleanLine)
		if len(parts) == 0 {
			continue
		}

		branchName := parts[0]
		isRemote := strings.HasPrefix(branchName, "remotes/")

		hash := ""
		if len(parts) > 1 {
			hash = parts[1]
		}

		msg := ""
		if len(parts) > 2 {
			msg = strings.Join(parts[2:], " ")
		}

		branches = append(branches, BranchInfo{
			Name:      branchName,
			IsCurrent: isCurrent,
			IsRemote:  isRemote,
			Hash:      hash,
			Message:   msg,
		})
	}

	return branches, nil
}

// GetRemotes returns all configured remote repositories.
func GetRemotes() ([]RemoteInfo, error) {
	if !IsGitRepo() {
		return nil, fmt.Errorf("current directory is not a Git repository")
	}

	cmd := exec.Command("git", "remote", "-v")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list remotes: %w", err)
	}

	var remotes []RemoteInfo
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			t := strings.Trim(fields[2], "()")
			remotes = append(remotes, RemoteInfo{
				Name: fields[0],
				URL:  fields[1],
				Type: t,
			})
		}
	}

	return remotes, nil
}

// AddRemote runs `git remote add <name> <url>`
func AddRemote(name, url string) error {
	cmd := exec.Command("git", "remote", "add", name, url)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// PromptAddRemote interactively asks the user to add a remote.
func PromptAddRemote() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("\n\033[33mNo remote repository configured. Would you like to add one? (y/n): \033[0m")
	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))

	if answer != "y" && answer != "yes" {
		fmt.Println("No remote added.")
		return nil
	}

	fmt.Print("Enter remote repository URL (e.g. https://github.com/user/repo.git): ")
	rawURL, _ := reader.ReadString('\n')
	repoURL := strings.TrimSpace(rawURL)

	if repoURL == "" {
		return fmt.Errorf("repository URL cannot be empty")
	}

	remoteName := "origin"
	if err := AddRemote(remoteName, repoURL); err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}

	fmt.Printf("\n\033[1;32m✓ Successfully added remote '%s': %s\033[0m\n", remoteName, repoURL)
	fmt.Println("You can now push using: \033[1mgit push -u origin <branch>\033[0m")
	fmt.Println()

	return nil
}
