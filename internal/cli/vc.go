package cli

import (
	"fmt"

	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/vc"
	"github.com/spf13/cobra"
)

var vcStatusFlag bool

// ensureGrydInitialized verifies that the current directory belongs to an initialized Gryd project.
func ensureGrydInitialized() error {
	_, err := config.FindConfigFile("", "")
	if err != nil {
		return fmt.Errorf("Gryd is not initialized in this project.\nRun 'gryd init' first to initialize Gryd before using version control commands")
	}
	return nil
}

var vcCmd = &cobra.Command{
	Use:   "vc [command]",
	Short: "Version control operations for Gryd projects",
	Long:  `Manage and inspect version control, Git repository status, branches, and remotes for Gryd projects.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return ensureGrydInitialized()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		info := vc.CheckGitStatus()
		termW := 80
		vc.PrintGitStatusCard(info, termW)
		return nil
	},
}

var vcStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check whether Git is installed and user is configured/logged in",
	Long:  `Verify Git installation, global user credentials, GitHub CLI status, and local repository branch.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		info := vc.CheckGitStatus()
		termW := 80
		vc.PrintGitStatusCard(info, termW)
		return nil
	},
}

var vcBranchesCmd = &cobra.Command{
	Use:   "branches",
	Short: "Display all local and remote branches in the current repository",
	Long:  `List branches cleanly with commit hashes, tracking status, and latest commit messages in a structured table.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !vc.IsGitRepo() {
			return fmt.Errorf("current directory is not inside a Git repository")
		}

		branches, err := vc.GetFormattedBranches()
		if err != nil {
			return err
		}

		if len(branches) == 0 {
			fmt.Println("No branches found in this repository.")
			return nil
		}

		vc.PrintBranchesTable(branches)
		return nil
	},
}

var vcRemoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Display or configure remote repositories",
	Long: `Display all configured remote repositories for the current Git repository.
If no remote exists, Gryd interactively prompts you to configure one.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !vc.IsGitRepo() {
			return fmt.Errorf("current directory is not inside a Git repository")
		}

		remotes, err := vc.GetRemotes()
		if err != nil {
			return err
		}

		if len(remotes) == 0 {
			return vc.PromptAddRemote()
		}

		fmt.Println("\n\033[1mConfigured Remote Repositories\033[0m")
		fmt.Println("───────────────────────────────────────────────────────")

		for _, r := range remotes {
			fmt.Printf("  • \033[1m%-10s\033[0m %-42s \033[2m(%s)\033[0m\n", r.Name, r.URL, r.Type)
		}
		fmt.Println()

		return nil
	},
}

func init() {
	vcCmd.Flags().BoolVar(&vcStatusFlag, "status", false, "Check Git installation and user login status")
	vcCmd.AddCommand(vcStatusCmd)
	vcCmd.AddCommand(vcBranchesCmd)
	vcCmd.AddCommand(vcRemoteCmd)
	vcCmd.AddCommand(vcCommitCmd)
	vcCmd.AddCommand(vcPushCmd)
	vcCmd.AddCommand(vcMergeCmd)
}
