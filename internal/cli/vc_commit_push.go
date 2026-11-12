package cli

import (
	"fmt"

	"github.com/loom-cli/gryd/internal/vc"
	"github.com/spf13/cobra"
)

var commitMessageFlag string

var vcCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Record changes to the repository with a commit message",
	Long: `Select a target branch (or create a new one), stage changes, and commit to the repository.
Prompts for a commit message or accepts one via the -m flag.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !vc.IsGitRepo() {
			return fmt.Errorf("current directory is not inside a Git repository")
		}

		hasChanges, err := vc.HasUncommittedChanges()
		if err != nil {
			return err
		}

		if !hasChanges {
			fmt.Println("No changes to commit (working tree clean).")
			return nil
		}

		// Step 1: Select branch or type a new one
		targetBranch, isNew, err := vc.SelectBranchPrompt()
		if err != nil {
			return err
		}

		if err := vc.SwitchOrCheckoutBranch(targetBranch, isNew); err != nil {
			return err
		}
		if isNew {
			fmt.Printf("✓ Created and switched to new branch: \033[1;32m%s\033[0m\n\n", targetBranch)
		} else {
			fmt.Printf("✓ On branch: \033[1;32m%s\033[0m\n\n", targetBranch)
		}

		// Step 2: Get commit message
		message := commitMessageFlag
		if message == "" {
			msg, err := vc.PromptForCommitMessage("")
			if err != nil {
				return err
			}
			message = msg
		}

		// Step 3: Stage all changes
		if err := vc.StageAll(); err != nil {
			return err
		}

		// Step 4: Commit
		out, err := vc.Commit(message)
		if err != nil {
			return err
		}

		fmt.Printf("\n\033[1;32m✓ Changes committed successfully on branch '%s'.\033[0m\n", targetBranch)
		fmt.Printf("%s\n\n", out)
		return nil
	},
}

var vcPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Commit any pending changes and push to a remote repository",
	Long: `Pushes to a remote repository on a selected branch.
Displays branches to choose from or lets you type a new branch name.
If uncommitted changes exist, Gryd commits them before pushing.
Presents an interactive selector with arrow keys to choose the remote.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !vc.IsGitRepo() {
			return fmt.Errorf("current directory is not inside a Git repository")
		}

		// Step 1: Branch selection first (select existing or type new)
		targetBranch, isNew, err := vc.SelectBranchPrompt()
		if err != nil {
			return err
		}

		if err := vc.SwitchOrCheckoutBranch(targetBranch, isNew); err != nil {
			return err
		}
		if isNew {
			fmt.Printf("✓ Created and switched to new branch: \033[1;32m%s\033[0m\n\n", targetBranch)
		} else {
			fmt.Printf("✓ On branch: \033[1;32m%s\033[0m\n\n", targetBranch)
		}

		// Step 2: Check for uncommitted changes
		hasChanges, err := vc.HasUncommittedChanges()
		if err != nil {
			return err
		}

		if hasChanges {
			fmt.Println("\033[1;33m[gryd vc] Uncommitted changes detected.\033[0m")
			msg, err := vc.PromptForCommitMessage("update")
			if err != nil {
				return err
			}

			if err := vc.StageAll(); err != nil {
				return err
			}

			commitOut, err := vc.Commit(msg)
			if err != nil {
				return err
			}

			fmt.Printf("\n\033[1;32m✓ Changes committed on '%s':\033[0m %s\n", targetBranch, msg)
			fmt.Println(commitOut)
		}

		// Step 3: Get remotes
		remotes, err := vc.GetRemotes()
		if err != nil {
			return err
		}

		if len(remotes) == 0 {
			if err := vc.PromptAddRemote(); err != nil {
				return err
			}
			remotes, _ = vc.GetRemotes()
			if len(remotes) == 0 {
				return fmt.Errorf("no remote available to push to")
			}
		}

		// Step 4: Interactive remote selection using arrow keys
		chosenRemote, err := vc.SelectRemote(remotes)
		if err != nil {
			return err
		}

		fmt.Printf("\n\033[1;36m[gryd vc] Pushing branch '%s' to remote '%s'...\033[0m\n\n", targetBranch, chosenRemote)

		// Step 5: Push
		if err := vc.Push(chosenRemote, targetBranch); err != nil {
			return fmt.Errorf("git push failed: %w", err)
		}

		fmt.Printf("\n\033[1;32m✓ Successfully pushed '%s' to '%s'.\033[0m\n\n", targetBranch, chosenRemote)
		return nil
	},
}

func init() {
	vcCommitCmd.Flags().StringVarP(&commitMessageFlag, "message", "m", "", "Commit message")
}
