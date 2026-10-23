// Package cmd contains all CLI commands for teacli, the non-interactive CLI for
// Gitea and Forgejo.
package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"

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
	hostName     string
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
	Version:       "1.0.2",
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
	RootCmd.PersistentFlags().StringVar(&hostName, "host", "", "Configured instance name, hostname, or URL (required with multiple instances)")
	RootCmd.PersistentFlags().StringVar(&serverName, "server", "", "Alias for --host")
	RootCmd.PersistentFlags().StringVar(&outputFormat, "format", "", "Output format: text (default), json")
	RootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	RootCmd.PersistentFlags().BoolVar(&fetchAll, "all", false, "Auto-paginate list commands and return the full set")
}

// getServer selects the only configured instance unless an explicit host is given.
func getServer(cmd *cobra.Command) (*config.ServerConfig, error) {
	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		return nil, errors.NewGeneralError(fmt.Sprintf("failed to load config: %v", err))
	}
	name, err := selectedServer(cfg)
	if err != nil {
		return nil, err
	}
	return cfg.GetServer(name)
}

func selectedServer(cfg *config.Config) (string, error) {
	if hostName != "" && serverName != "" {
		return "", errors.NewValidationError("pass only one of --host or --server", nil)
	}
	host := hostName
	if host == "" {
		host = serverName
	}
	names := cfg.ListServers()
	if host == "" {
		switch len(names) {
		case 0:
			return "", errors.NewValidationError("cannot detect host: no instances configured; use 'teacli auth login <server-url> --token <token>', then specify --host <name>", nil)
		case 1:
			return names[0], nil
		default:
			return "", errors.NewValidationError("multiple instances configured; specify --host <name>", nil)
		}
	}
	if _, err := cfg.GetServer(host); err == nil {
		return host, nil
	}
	var matched string
	for _, name := range names {
		server, err := cfg.GetServer(name)
		if err != nil {
			return "", err
		}
		u, err := url.Parse(server.URL)
		if err != nil {
			continue
		}
		if !strings.EqualFold(u.Host, host) && strings.TrimRight(server.URL, "/") != strings.TrimRight(host, "/") {
			continue
		}
		if matched != "" {
			return "", errors.NewValidationError("host matches multiple instances; specify --host <configured-name>", nil)
		}
		matched = name
	}
	if matched == "" {
		return "", errors.NewValidationError("unknown host; specify --host <configured-name> (see 'teacli auth config list')", nil)
	}
	return matched, nil
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
