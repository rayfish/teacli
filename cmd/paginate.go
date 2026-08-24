package cmd

import (
	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

// defaultPageSize is what we ask for when the command has no --per-page flag.
// The server clamps this to MAX_RESPONSE_ITEMS, which fetchAllPages detects.
const defaultPageSize = 50

// maxPages bounds auto-pagination so a misbehaving server cannot loop forever.
const maxPages = 1000

// listOptions builds ListOptions from the --page/--per-page flags, for commands
// that define them. Commands without those flags get the server defaults.
func listOptions(cmd *cobra.Command) gitea.ListOptions {
	opts := gitea.ListOptions{}
	if f := cmd.Flags().Lookup("page"); f != nil {
		opts.Page, _ = cmd.Flags().GetInt("page")
	}
	if f := cmd.Flags().Lookup("per-page"); f != nil {
		opts.PageSize, _ = cmd.Flags().GetInt("per-page")
	}
	return opts
}

// fetchList runs a paginated SDK list call. By default it returns a single page,
// honouring --page/--per-page. With --all it walks every page and returns the
// full set.
//
// The end of the list is detected by comparing each page against the size of the
// first page rather than against the requested size, because the server clamps
// PageSize to MAX_RESPONSE_ITEMS and would otherwise look like a short page.
func fetchList[T any](cmd *cobra.Command, fetch func(gitea.ListOptions) ([]T, *gitea.Response, error)) ([]T, error) {
	opts := listOptions(cmd)

	if !fetchAll {
		items, resp, err := fetch(opts)
		if err != nil {
			return nil, errors.FromGitea(resp, err)
		}
		return items, nil
	}

	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}

	var all []T
	effective := 0

	for page := 1; page <= maxPages; page++ {
		items, resp, err := fetch(gitea.ListOptions{Page: page, PageSize: pageSize})
		if err != nil {
			return nil, errors.FromGitea(resp, err)
		}
		all = append(all, items...)

		if len(items) == 0 {
			break
		}
		if page == 1 {
			effective = len(items)
		}
		if len(items) < effective {
			break
		}
	}

	return all, nil
}
