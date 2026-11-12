package vc

import (
	"fmt"
	"os/exec"
	"strings"
)

// SwitchOrCheckoutBranch switches to an existing branch or creates a new one,
// interactively resolving any uncommitted local change conflicts.
func SwitchOrCheckoutBranch(branchName string, create bool) error {
	branchName = strings.TrimSpace(branchName)
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	current, err := GetCurrentBranch()
	if err == nil && current == branchName {
		return nil // already on target branch
	}

	var cmd *exec.Cmd
	if create {
		cmd = exec.Command("git", "checkout", "-b", branchName)
	} else {
		cmd = exec.Command("git", "checkout", branchName)
	}

	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	outputStr := string(out)

	// Check if failure is due to uncommitted local changes conflicting with checkout
	isConflict := strings.Contains(outputStr, "Your local changes to the following files would be overwritten") ||
		strings.Contains(outputStr, "Please commit your changes or stash them")

	if !isConflict {
		action := "switch to"
		if create {
			action = "create"
		}
		return fmt.Errorf("failed to %s branch '%s': %s (%w)", action, branchName, strings.TrimSpace(outputStr), err)
	}

	// Interactive conflict resolution
	choice, promptErr := PromptBranchConflictResolution(current, branchName)
	if promptErr != nil || choice == "cancel" {
		return fmt.Errorf("branch switch aborted: local changes conflict with '%s'", branchName)
	}

	switch choice {
	case "stash-pop":
		fmt.Printf("\n[gryd vc] Stashing changes on '%s'...\n", current)
		if stashOut, err := exec.Command("git", "stash", "push", "-m", "gryd auto-stash before switching to "+branchName).CombinedOutput(); err != nil {
			return fmt.Errorf("git stash failed: %s (%w)", strings.TrimSpace(string(stashOut)), err)
		}

		fmt.Printf("[gryd vc] Switching to '%s'...\n", branchName)
		if err := switchDirect(branchName, create); err != nil {
			_ = exec.Command("git", "stash", "pop").Run() // attempt recovery
			return err
		}

		fmt.Printf("[gryd vc] Re-applying stashed changes on '%s'...\n", branchName)
		popOut, popErr := exec.Command("git", "stash", "pop").CombinedOutput()
		if popErr != nil {
			fmt.Printf("\033[1;33mNote: Stash applied with potential merge markers: %s\033[0m\n", strings.TrimSpace(string(popOut)))
		} else {
			fmt.Printf("\033[1;32m✓ Changes successfully restored on '%s'.\033[0m\n", branchName)
		}
		return nil

	case "commit":
		fmt.Printf("\n[gryd vc] Committing local changes to '%s' first...\n", current)
		msg, err := PromptForCommitMessage("WIP before switching to " + branchName)
		if err != nil {
			return err
		}
		if err := StageAll(); err != nil {
			return err
		}
		if cOut, err := Commit(msg); err != nil {
			return err
		} else {
			fmt.Printf("✓ Committed on '%s':\n%s\n", current, cOut)
		}

		fmt.Printf("[gryd vc] Switching to '%s'...\n", branchName)
		return switchDirect(branchName, create)

	case "stash-keep":
		fmt.Printf("\n[gryd vc] Stashing changes on '%s' into git stash...\n", current)
		if stashOut, err := exec.Command("git", "stash", "push", "-m", "gryd stash before "+branchName).CombinedOutput(); err != nil {
			return fmt.Errorf("git stash failed: %s (%w)", strings.TrimSpace(string(stashOut)), err)
		}
		fmt.Printf("✓ Stash saved. Switching to '%s'...\n", branchName)
		return switchDirect(branchName, create)

	case "force":
		fmt.Printf("\n\033[1;31m[gryd vc] Force switching to '%s' (discarding uncommitted modifications)...\033[0m\n", branchName)
		var fCmd *exec.Cmd
		if create {
			fCmd = exec.Command("git", "checkout", "-B", branchName)
		} else {
			fCmd = exec.Command("git", "checkout", "-f", branchName)
		}
		if fOut, err := fCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("force checkout failed: %s (%w)", strings.TrimSpace(string(fOut)), err)
		}
		fmt.Printf("✓ Successfully force checked out '%s'.\033[0m\n", branchName)
		return nil
	}

	return nil
}

func switchDirect(branchName string, create bool) error {
	var cmd *exec.Cmd
	if create {
		cmd = exec.Command("git", "checkout", "-b", branchName)
	} else {
		cmd = exec.Command("git", "checkout", branchName)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("checkout failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// GetLocalBranches returns all local and remote branches for branch selection.
func GetLocalBranches() ([]string, string, error) {
	current, _ := GetCurrentBranch()

	cmd := exec.Command("git", "branch", "-a", "--format=%(refname:short)")
	out, err := cmd.Output()
	if err != nil {
		return nil, current, fmt.Errorf("failed to list branches: %w", err)
	}

	seen := make(map[string]bool)
	var branches []string

	if current != "" {
		seen[current] = true
		branches = append(branches, current)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || strings.Contains(l, "/HEAD") {
			continue
		}
		cleanName := strings.TrimPrefix(l, "origin/")
		if !seen[cleanName] {
			seen[cleanName] = true
			branches = append(branches, cleanName)
		}
	}

	return branches, current, nil
}
