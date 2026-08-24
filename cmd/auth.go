// Package cmd contains authentication commands
package cmd

import (
	"fmt"
	"strings"

	"github.com/rayfish/teacli/modules/config"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication and configuration management",
	Long:  `Manage Gitea server authentication and configuration`,
}

var authLoginCmd = &cobra.Command{
	Use:     "login <server-url>",
	Aliases: []string{"add"},
	Short:   "Login to a Gitea server",
	Long: `Login to a Gitea server by storing credentials.

Examples:
  teacli auth login --token abc123 https://gitea.example.com
  teacli auth login --token abc123 https://gitea.example.com --name production
  teacli auth login --token abc123 https://gitea.example.com --default`,
	Args: cobra.ExactArgs(1),
	RunE: runAuthLogin,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout [server-name]",
	Short: "Logout from a Gitea server",
	Long: `Remove stored credentials for a server.
If no server is specified, removes the default server.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAuthLogout,
}

var authConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "View and manage configuration",
	Long:  `View and manage Gitea CLI configuration.`,
}

var authConfigListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured servers",
	RunE:  runConfigList,
}

var authConfigSetCmd = &cobra.Command{
	Use:   "set <server> <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(3),
	RunE:  runConfigSet,
}

var authConfigUnsetCmd = &cobra.Command{
	Use:   "unset <server>",
	Short: "Unset a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigUnset,
}

func init() {
	authLoginCmd.Flags().StringP("token", "t", "", "Gitea API token (required)")
	authLoginCmd.MarkFlagRequired("token")
	authLoginCmd.Flags().StringP("name", "n", "", "Server name for config (defaults to server hostname)")
	authLoginCmd.Flags().Bool("default", false, "Set as default server")

	authConfigSetCmd.Flags().Bool("default", false, "Set as default server")

	loginAliasCmd := &cobra.Command{
		Use:   "login <server-url>",
		Short: "Login to a Gitea server (alias for 'auth login')",
		Args:  cobra.ExactArgs(1),
		RunE:  runAuthLogin,
	}
	loginAliasCmd.Flags().StringP("token", "t", "", "Gitea API token (required)")
	loginAliasCmd.MarkFlagRequired("token")
	loginAliasCmd.Flags().StringP("name", "n", "", "Server name for config (defaults to server hostname)")
	loginAliasCmd.Flags().Bool("default", false, "Set as default server")

	logoutAliasCmd := &cobra.Command{
		Use:   "logout [server-name]",
		Short: "Logout from a Gitea server (alias for 'auth logout')",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runAuthLogout,
	}

	RootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd, authLogoutCmd, authConfigCmd)
	authConfigCmd.AddCommand(authConfigListCmd, authConfigSetCmd, authConfigUnsetCmd)
	RootCmd.AddCommand(loginAliasCmd, logoutAliasCmd)
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	serverURL := args[0]
	if serverURL == "" {
		return errors.NewValidationError("server URL is required",
			map[string]interface{}{"usage": "teacli auth login <server-url> --token <token>"})
	}

	token, _ := cmd.Flags().GetString("token")
	if token == "" {
		return errors.NewValidationError("token is required (--token)",
			map[string]interface{}{"missing_flags": []string{"--token"}})
	}

	cfg, err := config.Load()
	if err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to load config: %v", err))
	}

	serverName, _ := cmd.Flags().GetString("name")
	if serverName == "" {
		serverName = extractHostname(serverURL)
	}

	err = cfg.AddServer(serverName, serverURL, token)
	if err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to save config: %v", err))
	}

	isDefault, _ := cmd.Flags().GetBool("default")
	if isDefault || len(cfg.ListServers()) == 1 {
		err = cfg.SetDefaultServer(serverName)
		if err != nil {
			return errors.NewGeneralError(fmt.Sprintf("failed to set default server: %v", err))
		}
	}

	printer := getPrinter(cmd)
	setDefault := isDefault || len(cfg.ListServers()) == 1
	return printer.Emit(map[string]any{"status": "logged_in", "server": serverName, "url": serverURL, "default": setDefault}, func() error {
		printer.Printf("Logged in to %s as server '%s'\n", serverURL, serverName)

		if setDefault {
			printer.Println("Set as default server")
		}

		return nil
	})
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to load config: %v", err))
	}

	var serverName string
	if len(args) > 0 {
		serverName = args[0]
	} else {
		server, _ := cfg.GetDefaultServer()
		if server == nil {
			return errors.NewValidationError("no servers configured", nil)
		}
		serverName = extractHostname(server.URL)
	}

	err = cfg.RemoveServer(serverName)
	if err != nil {
		return errors.NewValidationError(fmt.Sprintf("failed to remove server: %v", err), nil)
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"status": "logged_out", "server": serverName}, func() error {
		printer.Printf("Logged out from server '%s'\n", serverName)
		return nil
	})
}

func runConfigList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to load config: %v", err))
	}

	printer := getPrinter(cmd)
	servers := cfg.ListServers()

	type ServerInfo struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Default bool   `json:"default"`
	}

	var data []ServerInfo
	defaultServer, _ := cfg.GetDefaultServer()

	for _, name := range servers {
		server, _ := cfg.GetServer(name)
		isDefault := defaultServer != nil && defaultServer.URL == server.URL

		data = append(data, ServerInfo{
			Name:    name,
			URL:     server.URL,
			Default: isDefault,
		})
	}

	headers := []string{"Name", "URL", "Default"}
	var rows [][]string
	for _, s := range data {
		rows = append(rows, []string{s.Name, s.URL, fmt.Sprintf("%t", s.Default)})
	}

	return printer.Emit(data, func() error {
		if len(data) == 0 {
			printer.Println("No servers configured")
			return nil
		}
		return printer.PrintTable(headers, rows)
	})
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to load config: %v", err))
	}

	serverName := args[0]
	key := args[1]
	value := args[2]

	if serverName == "" || key == "" || value == "" {
		return errors.NewValidationError("usage: teacli auth config set <server> <key> <value>",
			map[string]interface{}{"usage": "teacli auth config set <server> <key> <value>"})
	}

	server, err := cfg.GetServer(serverName)
	if err != nil {
		return errors.NewNotFoundError("server", serverName)
	}

	switch key {
	case "url":
		server.URL = value
	case "token":
		server.Token = value
	default:
		return errors.NewValidationError(fmt.Sprintf("unknown key: %s", key), nil)
	}

	err = cfg.Save()
	if err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to save config: %v", err))
	}

	isDefault, _ := cmd.Flags().GetBool("default")
	if isDefault {
		cfg.SetDefaultServer(serverName)
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"status": "updated", "server": serverName, "key": key}, func() error {
		printer.Printf("Updated server '%s' key '%s'\n", serverName, key)
		return nil
	})
}

func runConfigUnset(cmd *cobra.Command, args []string) error {
	return runAuthLogout(cmd, args)
}

func extractHostname(url string) string {
	host := url
	if len(host) > 8 && host[:8] == "https://" {
		host = host[8:]
	} else if len(host) > 7 && host[:7] == "http://" {
		host = host[7:]
	}
	for i, c := range host {
		if c == ':' {
			host = host[:i]
			break
		}
	}
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	return host
}
