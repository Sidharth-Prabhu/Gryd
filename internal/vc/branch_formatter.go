package vc

import (
	"fmt"
	"os/exec"
	"strings"
)

// BranchEntry holds structured information about a branch.
type BranchEntry struct {
	Name        string
	IsCurrent   bool
	IsRemote    bool
	CommitHash  string
	Upstream    string
	Description string
}

// GetFormattedBranches queries git for branches with explicit custom format.
func GetFormattedBranches() ([]BranchEntry, error) {
	if !IsGitRepo() {
		return nil, fmt.Errorf("current directory is not a Git repository")
	}

	// Format: %(HEAD)|%(refname:short)|%(objectname:short)|%(upstream:short)|%(subject)
	format := "%(HEAD)|%(refname:short)|%(objectname:short)|%(upstream:short)|%(subject)"
	cmd := exec.Command("git", "for-each-ref", "--format="+format, "refs/heads/", "refs/remotes/")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve branches: %w", err)
	}

	var entries []BranchEntry
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 3 {
			continue
		}

		head := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		hash := strings.TrimSpace(parts[2])

		upstream := ""
		if len(parts) > 3 {
			upstream = strings.TrimSpace(parts[3])
		}

		subject := ""
		if len(parts) > 4 {
			subject = strings.TrimSpace(parts[4])
		}

		// Skip symbolic ref HEAD pointers like origin/HEAD
		if strings.HasSuffix(name, "/HEAD") {
			continue
		}

		isRemote := strings.HasPrefix(name, "origin/") || strings.HasPrefix(name, "upstream/")
		isCurrent := (head == "*")

		entries = append(entries, BranchEntry{
			Name:        name,
			IsCurrent:   isCurrent,
			IsRemote:    isRemote,
			CommitHash:  hash,
			Upstream:    upstream,
			Description: subject,
		})
	}

	return entries, nil
}

// PrintBranchesTable renders branches in a clean, aligned, sectioned table.
func PrintBranchesTable(branches []BranchEntry) {
	var localBranches []BranchEntry
	var remoteBranches []BranchEntry

	for _, b := range branches {
		if b.IsRemote {
			remoteBranches = append(remoteBranches, b)
		} else {
			localBranches = append(localBranches, b)
		}
	}

	// 1. Local Branches Table
	fmt.Println("\n\033[1mLocal Branches\033[0m")
	fmt.Printf("  \033[1m%-32s %-8s %-24s %s\033[0m\n", "BRANCH", "COMMIT", "TRACKING", "LATEST COMMIT")
	fmt.Printf("  %s\n", strings.Repeat("─", 88))

	for _, b := range localBranches {
		indicator := "  "
		nameStyled := b.Name

		if b.IsCurrent {
			indicator = "\033[1;32m* \033[0m"
			nameStyled = fmt.Sprintf("\033[1;32m%-30s\033[0m", b.Name+" (HEAD)")
		} else {
			nameStyled = fmt.Sprintf("%-30s", b.Name)
		}

		trackingStr := "-"
		if b.Upstream != "" {
			trackingStr = fmt.Sprintf("\033[36m%s\033[0m", b.Upstream)
		}

		desc := b.Description
		if len(desc) > 36 {
			desc = desc[:33] + "..."
		}

		fmt.Printf("%s%-30s \033[2m%-8s\033[0m %-33s %s\n",
			indicator, nameStyled, b.CommitHash, trackingStr, desc)
	}

	// 2. Remote Branches
	if len(remoteBranches) > 0 {
		fmt.Printf("\n\033[1mRemote Tracking Branches (%d)\033[0m\n", len(remoteBranches))
		fmt.Printf("  \033[1m%-38s %-8s %s\033[0m\n", "REMOTE BRANCH", "COMMIT", "LATEST COMMIT")
		fmt.Printf("  %s\n", strings.Repeat("─", 88))

		for _, b := range remoteBranches {
			desc := b.Description
			if len(desc) > 42 {
				desc = desc[:39] + "..."
			}

			fmt.Printf("  \033[2m%-38s %-8s\033[0m %s\n",
				b.Name, b.CommitHash, desc)
		}
	}

	fmt.Println()
}
