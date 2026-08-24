// Package main is the entry point for the non-interactive Gitea CLI
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/rayfish/teacli/cmd"
	"github.com/rayfish/teacli/modules/errors"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("teacli: ")

	if err := cmd.Execute(); err != nil {
		code := errors.ExitCode(err)

		// Errors follow --format like everything else: readable by default,
		// machine-parsable only when JSON was asked for.
		if cmd.ResolvedFormat() == "json" {
			fmt.Fprintf(os.Stderr, "{\"error\":true,\"code\":%d,\"message\":%q}\n", code, errors.Message(err))
		} else {
			log.Println(errors.Message(err))
		}

		os.Exit(code)
	}
}
