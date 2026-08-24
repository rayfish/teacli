package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func findCmd(root *cobra.Command, path ...string) *cobra.Command {
	cur := root
	for _, name := range path {
		next, _, err := cur.Find([]string{name})
		if err != nil || next == cur {
			return nil
		}
		cur = next
	}
	return cur
}

func TestGlobalFlagsArePersistent(t *testing.T) {
	sub := findCmd(RootCmd, "pr", "list")
	if sub == nil {
		t.Fatal("pr list command not found")
	}
	for _, name := range []string{"format", "server", "dry-run", "all"} {
		// cobra only merges persistent flags into a command's own FlagSet
		// lazily (e.g. during Execute/ParseFlags), so Find() alone leaves
		// sub.Flags() unmerged. Command.Flag() is cobra's public API for
		// checking local-or-inherited flags without relying on that lazy
		// merge, which is what happens during real command execution.
		if sub.Flag(name) == nil {
			t.Errorf("subcommand 'pr list' does not inherit --%s", name)
		}
	}
}

func TestGlobalPersistentFlagsHaveNoShorthand(t *testing.T) {
	RootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Shorthand != "" {
			t.Errorf("persistent global flag --%s has shorthand -%s; global flags must be long-form only to avoid subcommand shorthand collisions", f.Name, f.Shorthand)
		}
	})
}

func TestRootUseIsTeacli(t *testing.T) {
	if RootCmd.Use != "teacli" {
		t.Errorf("RootCmd.Use = %q, want \"teacli\"", RootCmd.Use)
	}
}
