package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Commands that take a repository but deliberately do not infer one, with the
// reason they are excluded.
var noInference = map[string]string{
	"teacli notification list":    "omitting the repository already means every repository",
	"teacli notification read":    "omitting the repository already means every repository",
	"teacli repo delete":          "deleting is permanent and nothing prompts, so it stays explicit",
	"teacli admin adopt":          "an unadopted repository has no clone to stand in",
	"teacli admin drop-unadopted": "an unadopted repository has no clone to stand in",
}

const repoSlot = "<owner>/<repo>"

// positionals returns the argument tokens of a Use line, without the command
// path that precedes them.
func positionals(use string) []string {
	fields := strings.Fields(use)
	if len(fields) == 0 {
		return nil
	}
	return fields[1:]
}

// requiredAfterRepo counts the arguments a command needs besides the
// repository. A bracketed token is optional and a variadic one counts once.
// Some Use lines go on to document a mandatory flag, which is where the
// positionals stop.
func requiredAfterRepo(tokens []string) int {
	n := 0
	for _, tok := range tokens {
		if strings.HasPrefix(tok, "-") {
			break
		}
		if strings.Contains(tok, repoSlot) || strings.HasPrefix(tok, "[") {
			continue
		}
		n++
	}
	return n
}

// walk visits every runnable command in the tree.
func walk(cmd *cobra.Command, visit func(*cobra.Command)) {
	if cmd.Runnable() {
		visit(cmd)
	}
	for _, child := range cmd.Commands() {
		walk(child, visit)
	}
}

func TestEveryLeadingRepositoryArgumentIsOptional(t *testing.T) {
	walk(RootCmd, func(cmd *cobra.Command) {
		tokens := positionals(cmd.Use)
		if len(tokens) == 0 || !strings.Contains(tokens[0], repoSlot) {
			return
		}
		path := cmd.CommandPath()
		if _, skip := noInference[path]; skip {
			return
		}

		if tokens[0] != "["+repoSlot+"]" {
			t.Errorf("%s: Use has %q, want the repository shown as optional %q",
				path, tokens[0], "["+repoSlot+"]")
		}

		want := requiredAfterRepo(tokens)
		if cmd.Args == nil {
			t.Errorf("%s: no Args validator", path)
			return
		}
		if err := cmd.Args(cmd, make([]string, want)); err != nil {
			t.Errorf("%s: rejects %d argument(s) with the repository omitted: %v", path, want, err)
		}
		if err := cmd.Args(cmd, make([]string, want+1)); err != nil {
			t.Errorf("%s: rejects %d argument(s) with the repository given: %v", path, want+1, err)
		}
	})
}
