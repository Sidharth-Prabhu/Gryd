package vc

import (
	"fmt"
	"os/exec"
	"strings"
)

// MergeBranches runs git merge <sourceBranch> into the current branch.
func MergeBranches(sourceBranch string, noFF bool, squash bool) (string, error) {
	args := []string{"merge"}
	if noFF {
		args = append(args, "--no-ff")
	}
	if squash {
		args = append(args, "--squash")
	}
	args = append(args, sourceBranch)

	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	outStr := strings.TrimSpace(string(out))

	if err != nil {
		// Check for merge conflicts
		conflicts := GetConflictedFiles()
		if len(conflicts) > 0 {
			return outStr, fmt.Errorf("merge conflict detected in %d file(s): %s",
				len(conflicts), strings.Join(conflicts, ", "))
		}
		return outStr, fmt.Errorf("%s (%w)", outStr, err)
	}

	return outStr, nil
}

// GetConflictedFiles lists unmerged files with conflicts.
func GetConflictedFiles() []string {
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=U")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var conflicts []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			conflicts = append(conflicts, l)
		}
	}
	return conflicts
}

// PromptSelectBranch displays an interactive menu to choose a branch with a custom prompt title.
func PromptSelectBranch(title string, defaultBranch string) (string, error) {
	branches, current, err := GetLocalBranches()
	if err != nil {
		return "", err
	}

	if defaultBranch != "" {
		current = defaultBranch
	}

	initialCursor := 0
	for i, b := range branches {
		if b == current {
			initialCursor = i
			break
		}
	}

	p := newBranchSelectorModel(branches, current)
	p.currentBranch = current
	p.cursor = initialCursor

	// Use bubbletea to run
	chosenBranch, isNew, err := SelectBranchPrompt()
	if err != nil {
		return "", err
	}
	_ = isNew
	return chosenBranch, nil
}
