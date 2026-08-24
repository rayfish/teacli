package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type schemaFlag struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand,omitempty"`
	Type      string `json:"type"`
	Default   string `json:"default,omitempty"`
	Usage     string `json:"usage,omitempty"`
}

type schemaCommand struct {
	Path  string       `json:"path"`
	Short string       `json:"short"`
	Use   string       `json:"use"`
	Flags []schemaFlag `json:"flags"`
}

type cliSchema struct {
	Name     string          `json:"name"`
	Version  string          `json:"version"`
	Commands []schemaCommand `json:"commands"`
}

func buildSchema(root *cobra.Command) cliSchema {
	s := cliSchema{Name: root.Name(), Version: root.Version}
	var walk func(c *cobra.Command, prefix string)
	walk = func(c *cobra.Command, prefix string) {
		for _, sub := range c.Commands() {
			if sub.Hidden || sub.Name() == "help" {
				continue
			}
			path := strings.TrimSpace(prefix + " " + sub.Name())
			if sub.Runnable() {
				sc := schemaCommand{Path: path, Short: sub.Short, Use: sub.Use}
				seen := make(map[string]bool)
				addFlag := func(f *pflag.Flag) {
					if seen[f.Name] {
						return
					}
					seen[f.Name] = true
					sc.Flags = append(sc.Flags, schemaFlag{
						Name: f.Name, Shorthand: f.Shorthand, Type: f.Value.Type(),
						Default: f.DefValue, Usage: f.Usage,
					})
				}
				// Flags() merges inherited persistent flags into the local set,
				// but call InheritedFlags() explicitly too so schema stays correct
				// even if cobra's merge-on-access behavior changes.
				sub.Flags().VisitAll(addFlag)
				sub.InheritedFlags().VisitAll(addFlag)
				s.Commands = append(s.Commands, sc)
			}
			walk(sub, path)
		}
	}
	walk(root, "")
	return s
}

var schemaCmd = &cobra.Command{
	Use:     "schema",
	Aliases: []string{"commands"},
	Short:   "Emit the full command tree as JSON for agent discovery",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		schema := buildSchema(RootCmd)
		printer := getPrinter(cmd)
		return printer.Emit(schema, func() error {
			for _, c := range schema.Commands {
				fmt.Printf("%s\t%s\n", c.Path, c.Short)
			}
			return nil
		})
	},
}

func init() {
	RootCmd.AddCommand(schemaCmd)
}
