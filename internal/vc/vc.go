package vc

import (
	"fmt"
	"os/exec"
	"strings"
)

// GitStatusInfo encapsulates system git installation and authentication state.
type GitStatusInfo struct {
	Installed       bool
	Version         string
	UserName        string
	UserEmail       string
	IsConfigured    bool
	GHInstalled     bool
	GHAuthenticated bool
	GHUser          string
	InGitRepo       bool
	RepoBranch      string
	RepoOrigin      string
}

// CheckGitStatus checks if git is installed, configured, and logged in.
func CheckGitStatus() GitStatusInfo {
	var info GitStatusInfo

	// 1. Check if git is installed
	versionOut, err := exec.Command("git", "--version").Output()
	if err != nil {
		info.Installed = false
		return info
	}

	info.Installed = true
	info.Version = strings.TrimSpace(string(versionOut))

	// 2. Check git identity (user.name & user.email)
	nameOut, _ := exec.Command("git", "config", "--get", "user.name").Output()
	emailOut, _ := exec.Command("git", "config", "--get", "user.email").Output()

	info.UserName = strings.TrimSpace(string(nameOut))
	info.UserEmail = strings.TrimSpace(string(emailOut))

	if info.UserName != "" && info.UserEmail != "" {
		info.IsConfigured = true
	}

	// 3. Check GitHub CLI authentication if installed
	if ghPath, err := exec.LookPath("gh"); err == nil && ghPath != "" {
		info.GHInstalled = true
		ghOut, err := exec.Command("gh", "auth", "status").CombinedOutput()
		outStr := string(ghOut)
		if err == nil || strings.Contains(outStr, "Logged in to") {
			info.GHAuthenticated = true
			// Extract username if present
			for _, line := range strings.Split(outStr, "\n") {
				if strings.Contains(line, "Logged in to") {
					parts := strings.Fields(line)
					for i, p := range parts {
						if p == "account" && i+1 < len(parts) {
							info.GHUser = parts[i+1]
							break
						}
					}
				}
			}
		}
	}

	// 4. Check if current directory is inside a git repo
	branchOut, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err == nil {
		info.InGitRepo = true
		info.RepoBranch = strings.TrimSpace(string(branchOut))
		originOut, err := exec.Command("git", "config", "--get", "remote.origin.url").Output()
		if err == nil {
			info.RepoOrigin = strings.TrimSpace(string(originOut))
		}
	}

	return info
}

// PrintGitStatus displays the status results in a clean format.
func PrintGitStatus(info GitStatusInfo) {
	fmt.Println("\n\033[1mVersion Control System Status\033[0m")
	fmt.Printf("%s\n\n", strings.Repeat("─", 50))

	if !info.Installed {
		fmt.Printf("\033[1;31m✗ Git is not installed on this system.\033[0m\n\n")
		fmt.Println("Please install Git first before using version control features.")
		fmt.Println("Installation instructions:")
		fmt.Println("  • macOS (Homebrew): brew install git")
		fmt.Println("  • Debian/Ubuntu:    sudo apt update && sudo apt install git")
		fmt.Println("  • Official Site:    https://git-scm.com/downloads")
		fmt.Println()
		return
	}

	fmt.Printf("  \033[1;32m✓\033[0m \033[1mGit Installed:\033[0m       %s\n", info.Version)

	if info.IsConfigured {
		fmt.Printf("  \033[1;32m✓\033[0m \033[1mUser Identity:\033[0m       %s <%s>\n", info.UserName, info.UserEmail)
	} else {
		fmt.Printf("  \033[1;31m✗\033[0m \033[1mUser Identity:\033[0m       Not configured\n")
		fmt.Println("\n  \033[33mPlease configure your Git identity:\033[0m")
		fmt.Println("    git config --global user.name 'Your Name'")
		fmt.Println("    git config --global user.email 'you@example.com'")
	}

	if info.GHInstalled {
		if info.GHAuthenticated {
			ghUser := info.GHUser
			if ghUser == "" {
				ghUser = "Authenticated"
			}
			fmt.Printf("  \033[1;32m✓\033[0m \033[1mGitHub CLI (gh):\033[0m     Logged in as %s\n", ghUser)
		} else {
			fmt.Printf("  \033[1;33m!\033[0m \033[1mGitHub CLI (gh):\033[0m     Not authenticated (run 'gh auth login')\n")
		}
	}

	if info.InGitRepo {
		fmt.Println()
		fmt.Printf("  \033[1mCurrent Repository:\033[0m\n")
		fmt.Printf("    • Branch: %s\n", info.RepoBranch)
		if info.RepoOrigin != "" {
			fmt.Printf("    • Remote: %s\n", info.RepoOrigin)
		}
	}

	fmt.Println()
}
