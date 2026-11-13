package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/logger"
	"github.com/loom-cli/gryd/internal/state"
	"github.com/spf13/cobra"
)

var (
	logsFollow bool
	logsTail   int
	logsOutput string
)

var logsCmd = &cobra.Command{
	Use:   "logs [service]",
	Short: "View, filter, or export service logs",
	Long: `Display logs for all services or filter by a specific service.
Supports live following (-f), tailing the last N lines (-n), and exporting to a file (-o).

Examples:
  gryd logs
  gryd logs backend
  gryd logs backend -f
  gryd logs worker --tail=50 -o worker.log`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName, err := config.DetectCurrentProject()
		if err != nil {
			running, _ := state.ListRunningProjects()
			if len(running) == 1 {
				projectName = running[0].ProjectName
			} else {
				return fmt.Errorf("could not detect project: %w", err)
			}
		}

		projectState, err := state.LoadState(projectName)
		if err != nil {
			return fmt.Errorf("failed to load state for project '%s': %w", projectName, err)
		}

		logFilePath := projectState.LogFilePath
		file, err := os.Open(logFilePath)
		if err != nil {
			return fmt.Errorf("failed to open log file (%s): %w", logFilePath, err)
		}
		defer file.Close()

		var targetService string
		if len(args) > 0 {
			targetService = args[0]
		}

		var outWriter io.Writer = os.Stdout
		if logsOutput != "" {
			outFile, err := os.Create(logsOutput)
			if err != nil {
				return fmt.Errorf("failed to create output file '%s': %w", logsOutput, err)
			}
			defer outFile.Close()
			outWriter = io.MultiWriter(os.Stdout, outFile)
		}

		// Read existing lines
		var lines []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if matchesServiceFilter(line, targetService) {
				lines = append(lines, line)
			}
		}

		if logsTail > 0 && len(lines) > logsTail {
			lines = lines[len(lines)-logsTail:]
		}

		for _, l := range lines {
			if logsOutput != "" {
				fmt.Fprintln(outWriter, l)
			} else {
				fmt.Fprintln(outWriter, logger.ColorizeLogLevels(l))
			}
		}

		if !logsFollow {
			if logsOutput != "" {
				fmt.Printf("\n\033[1;32m✓ Logs exported to %s\033[0m\n", logsOutput)
			}
			return nil
		}

		// Follow mode (-f)
		reader := bufio.NewReader(file)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				break
			}
			line = strings.TrimRight(line, "\r\n")
			if matchesServiceFilter(line, targetService) {
				if logsOutput != "" {
					fmt.Fprintln(outWriter, line)
				} else {
					fmt.Fprintln(outWriter, logger.ColorizeLogLevels(line))
				}
			}
		}

		return nil
	},
}

func matchesServiceFilter(line, service string) bool {
	if service == "" {
		return true
	}
	prefix := "[" + service + "]"
	return strings.HasPrefix(strings.TrimSpace(line), prefix)
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output (live stream)")
	logsCmd.Flags().IntVarP(&logsTail, "tail", "n", 0, "Number of lines to show from the end of the logs")
	logsCmd.Flags().StringVarP(&logsOutput, "output", "o", "", "Export logs to a file")
}
