package cli

import (
	"fmt"
	"strings"

	"github.com/loom-cli/gryd/internal/vc"
	"github.com/spf13/cobra"
)

var (
	mergeNoFF   bool
	mergeSquash bool
)

var vcMergeCmd = &cobra.Command{
	Use:   "merge [source_branch] [base_branch]",
	Short: "Merge branches into the base branch",
	Long: `Merge a source branch into a target base branch (e.g. 'main', 'master', 'development').
If branch names are omitted, Gryd interactively prompts you to choose the source branch
and the destination base branch using arrow keys.

Checks for uncommitted working tree changes before initiating the merge.`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !vc.IsGitRepo() {
			return fmt.Errorf("current directory is not inside a Git repository")
		}

		currentBranch, err := vc.GetCurrentBranch()
		if err != nil {
			return err
		}

		// Check for uncommitted changes
		hasChanges, err := vc.HasUncommittedChanges()
		if err != nil {
			return err
		}
		if hasChanges {
			return fmt.Errorf("you have uncommitted changes. Please commit or stash them before merging")
		}

		var sourceBranch string
		var baseBranch string

		if len(args) == 2 {
			sourceBranch = args[0]
			baseBranch = args[1]
		} else if len(args) == 1 {
			sourceBranch = args[0]
			// Prompt for base branch
			fmt.Printf("\nSelect the \033[1mbase branch\033[0m to merge '%s' into:\n", sourceBranch)
			chosenBase, _, err := vc.SelectBranchPrompt()
			if err != nil {
				return err
			}
			baseBranch = chosenBase
		} else {
			// Interactive mode:
			// Step 1: Select source branch
			fmt.Printf("\nSelect the \033[1msource branch\033[0m (branch with changes to merge):\n")
			chosenSource, _, err := vc.SelectBranchPrompt()
			if err != nil {
				return err
			}
			sourceBranch = chosenSource

			// Step 2: Select target base branch
			fmt.Printf("\nSelect the \033[1mbase branch\033[0m (target branch to merge into):\n")
			chosenBase, _, err := vc.SelectBranchPrompt()
			if err != nil {
				return err
			}
			baseBranch = chosenBase
		}

		if sourceBranch == baseBranch {
			return fmt.Errorf("cannot merge branch '%s' into itself", sourceBranch)
		}

		// Switch to base branch
		fmt.Printf("\n[gryd vc] Switching to base branch '\033[1m%s\033[0m'...\n", baseBranch)
		if err := vc.SwitchOrCheckoutBranch(baseBranch, false); err != nil {
			return err
		}

		// Perform merge
		fmt.Printf("[gryd vc] Merging '\033[1;36m%s\033[0m' into '\033[1;32m%s\033[0m'...\n\n", sourceBranch, baseBranch)
		out, err := vc.MergeBranches(sourceBranch, mergeNoFF, mergeSquash)
		if err != nil {
			fmt.Printf("\033[1;31m✗ Merge failed:\033[0m %v\n\n", err)
			conflicts := vc.GetConflictedFiles()
			if len(conflicts) > 0 {
				fmt.Println("\033[1;33mConflicted files:\033[0m")
				for _, f := range conflicts {
					fmt.Printf("  • %s\n", f)
				}
				fmt.Println("\nResolve conflicts in the listed files and run: \033[1mgryd vc commit\033[0m")
			}
			return err
		}

		if strings.TrimSpace(out) != "" {
			fmt.Println(out)
			fmt.Println()
		}

		fmt.Printf("\033[1;32m✓ Successfully merged '%s' into '%s'!\033[0m\n", sourceBranch, baseBranch)
		fmt.Printf("You can now push the updated base branch using: \033[1mgryd vc push\033[0m\n\n")

		// If user originally was on source branch and merged into another, inform them they are now on base branch
		_ = currentBranch
		return nil
	},
}

func init() {
	vcMergeCmd.Flags().BoolVar(&mergeNoFF, "no-ff", false, "Create a merge commit even when the merge resolves as a fast-forward")
	vcMergeCmd.Flags().BoolVar(&mergeSquash, "squash", false, "Produce the working tree and index state as if a real merge happened, but do not make a commit")
}
