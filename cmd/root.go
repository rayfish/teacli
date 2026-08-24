// Package cmd contains all CLI commands for teacli, the non-interactive CLI for
// Gitea and Forgejo.
package cmd

import (
	"fmt"
	"os"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/config"
	"github.com/rayfish/teacli/modules/errors"
	"github.com/rayfish/teacli/modules/output"

	"github.com/spf13/cobra"
)

// Global flags
var (
	outputFormat string
	dryRun       bool
	verbose      bool
	serverName   string
	configPath   string
	fetchAll     bool
)

// RootCmd is the root cobra command
var RootCmd = &cobra.Command{
	Use:   "teacli",
	Short: "Non-interactive CLI for Gitea and Forgejo, built for AI agents and automation",
	Long: `A fully non-interactive command-line tool for Gitea and Forgejo.

Nothing prompts, output is text by default and JSON on request, and every error
maps onto a documented exit code, so it works the same in a terminal, in a CI
job, and in the hands of an AI agent.`,
	Version:       "1.0.0",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// Execute runs the root command
func Execute() error {
	return RootCmd.Execute()
}

func init() {
	RootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Config file path")
	RootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Verbose output")
	RootCmd.PersistentFlags().StringVar(&serverName, "server", "", "Server name from config")
	RootCmd.PersistentFlags().StringVar(&outputFormat, "format", "", "Output format: text (default), json")
	RootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	RootCmd.PersistentFlags().BoolVar(&fetchAll, "all", false, "Auto-paginate list commands and return the full set")
}

// getServer resolves the server to talk to: --server if given, otherwise the default
func getServer(cmd *cobra.Command) (*config.ServerConfig, error) {
	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		return nil, errors.NewGeneralError(fmt.Sprintf("failed to load config: %v", err))
	}

	var server *config.ServerConfig

	if name := serverChoice(serverName, inferredServer()); name != "" {
		server, err = cfg.GetServer(name)
		if err != nil {
			return nil, errors.NewNotFoundError("server", name)
		}
	} else {
		server, err = cfg.GetDefaultServer()
		if err != nil || server == nil {
			return nil, errors.NewValidationError("no default server configured",
				map[string]interface{}{"hint": "use 'teacli auth login' or '--server <name>'"})
		}
	}

	return server, nil
}

// serverChoice reports which configured server to talk to: --server when it was
// given, then the server the inferred repository belongs to, and otherwise ""
// for the default. An explicit owner/repo never infers a server, so a command
// that names its repository is not redirected by the directory it runs in.
func serverChoice(flag, inferred string) string {
	if flag != "" {
		return flag
	}
	return inferred
}

// getClient creates a Gitea API client
func getClient(cmd *cobra.Command) (*gitea.Client, error) {
	server, err := getServer(cmd)
	if err != nil {
		return nil, err
	}

	client, err := gitea.NewClient(server.URL, gitea.SetToken(server.Token))
	if err != nil {
		return nil, errors.NewGeneralError(fmt.Sprintf("failed to create client: %v", err))
	}

	return client, nil
}

// ResolvedFormat reports the output format after flag parsing, using the same
// precedence as getPrinter. main uses it to render errors in the format the
// caller asked for.
func ResolvedFormat() string {
	if outputFormat != "" {
		return outputFormat
	}
	if env := os.Getenv("TEACLI_FORMAT"); env != "" {
		return env
	}
	return "text"
}

// getPrinter returns an output printer. Precedence: --format flag, then TEACLI_FORMAT
// env, then text (the default).
func getPrinter(cmd *cobra.Command) *output.Printer {
	format := ""
	if f, err := cmd.Flags().GetString("format"); err == nil {
		format = f
	}
	if format == "" {
		format = os.Getenv("TEACLI_FORMAT")
	}
	if format == "" {
		format = "text"
	}

	var p *output.Printer
	switch format {
	case "json":
		p = output.NewPrinter(output.FormatJSON)
	default:
		p = output.NewPrinter(output.FormatText)
	}
	if d, err := cmd.Flags().GetBool("dry-run"); err == nil {
		p.SetDryRun(d)
	}
	return p
}
