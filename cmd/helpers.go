package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/internal/utils"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

// repoArg parses an "owner/repo" argument into its parts, reporting a
// validation error (exit 2) rather than a generic failure when it is malformed.
func repoArg(arg string) (string, string, error) {
	owner, repo, err := utils.ParseRepoArg(arg)
	if err != nil {
		return "", "", errors.NewValidationError(err.Error(), map[string]interface{}{"arg": arg})
	}
	return owner, repo, nil
}

// int64Arg parses a numeric argument, naming the argument in the error so the
// caller knows which one was wrong.
func int64Arg(arg, name string) (int64, error) {
	n, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return 0, errors.NewValidationError(fmt.Sprintf("invalid %s: %s", name, arg),
			map[string]interface{}{"expected": "a number"})
	}
	return n, nil
}

// isDryRun reports whether --dry-run was given.
func isDryRun(cmd *cobra.Command) bool {
	d, _ := cmd.Flags().GetBool("dry-run")
	return d
}

// dryRunf prints what a mutating command would have done and reports true, so
// callers can `if dryRunf(...) { return nil }`.
func dryRunf(cmd *cobra.Command, format string, args ...any) bool {
	if !isDryRun(cmd) {
		return false
	}
	getPrinter(cmd).Printf("DRY-RUN: "+format+"\n", args...)
	return true
}

// addPageFlags gives a list command the per-command pagination flags that
// listOptions and fetchList read.
func addPageFlags(cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().Int("page", 0, "Page number")
		c.Flags().Int("per-page", 0, "Results per page")
	}
}

// emitMessage prints a one-line confirmation in text mode and a small JSON
// object under --format json, for mutations with no useful response body.
func emitMessage(cmd *cobra.Command, payload any, format string, args ...any) error {
	printer := getPrinter(cmd)
	return printer.Emit(payload, func() error {
		printer.Printf(format+"\n", args...)
		return nil
	})
}

// okMessage is the JSON payload for a mutation that returns nothing.
func okMessage(message string) map[string]any {
	return map[string]any{"ok": true, "message": message}
}

// parseState maps a --state value onto the SDK's state type.
func parseState(s string) (gitea.StateType, error) {
	switch s {
	case "", "open":
		return gitea.StateOpen, nil
	case "closed":
		return gitea.StateClosed, nil
	case "all":
		return gitea.StateAll, nil
	default:
		return "", errors.NewValidationError(fmt.Sprintf("invalid state: %s", s),
			map[string]interface{}{"expected": "open, closed, or all"})
	}
}

// readAllStdin reads the whole of stdin, for flags that accept "-".
func readAllStdin() ([]byte, error) {
	return io.ReadAll(os.Stdin)
}

// bodyFromFlags resolves a text body given as --<textFlag> or read from
// --<fileFlag>, where "-" means stdin. It returns an empty string when neither
// flag was given, leaving the caller to decide whether that is an error.
func bodyFromFlags(cmd *cobra.Command, textFlag, fileFlag string) (string, error) {
	text, _ := cmd.Flags().GetString(textFlag)
	file, _ := cmd.Flags().GetString(fileFlag)

	if text != "" && file != "" {
		return "", errors.NewValidationError(
			fmt.Sprintf("--%s and --%s cannot be combined", textFlag, fileFlag), nil)
	}
	if file == "" {
		return text, nil
	}

	var (
		data []byte
		err  error
	)
	if file == "-" {
		data, err = readAllStdin()
	} else {
		data, err = os.ReadFile(file)
	}
	if err != nil {
		return "", errors.NewValidationError(fmt.Sprintf("failed to read %s: %v", file, err), nil)
	}
	return string(data), nil
}

// yesNo renders a bool for table cells, where "true"/"false" reads worse than
// a short marker.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
