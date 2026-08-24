package cmd

import (
	"testing"

	"code.gitea.io/sdk/gitea"

	"github.com/spf13/cobra"
)

// pagedFetcher serves `total` items in pages of at most `serverMax`, which is how
// Gitea behaves once MAX_RESPONSE_ITEMS clamps a larger requested PageSize.
func pagedFetcher(total, serverMax int, calls *[]gitea.ListOptions) func(gitea.ListOptions) ([]int, *gitea.Response, error) {
	return func(lo gitea.ListOptions) ([]int, *gitea.Response, error) {
		*calls = append(*calls, lo)

		size := lo.PageSize
		if size <= 0 || size > serverMax {
			size = serverMax
		}
		page := lo.Page
		if page <= 0 {
			page = 1
		}

		start := (page - 1) * size
		if start >= total {
			return nil, nil, nil
		}
		end := start + size
		if end > total {
			end = total
		}

		items := make([]int, 0, end-start)
		for i := start; i < end; i++ {
			items = append(items, i)
		}
		return items, nil, nil
	}
}

func listCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "list"}
	cmd.Flags().Int("page", 0, "")
	cmd.Flags().Int("per-page", 0, "")
	return cmd
}

func TestFetchListSinglePageByDefault(t *testing.T) {
	fetchAll = false

	var calls []gitea.ListOptions
	items, err := fetchList(listCmd(), pagedFetcher(95, 30, &calls))
	if err != nil {
		t.Fatalf("fetchList: %v", err)
	}

	if len(items) != 30 {
		t.Errorf("got %d items, want a single page of 30", len(items))
	}
	if len(calls) != 1 {
		t.Errorf("made %d requests, want 1", len(calls))
	}
}

func TestFetchListAllWalksEveryPage(t *testing.T) {
	fetchAll = true
	defer func() { fetchAll = false }()

	var calls []gitea.ListOptions
	items, err := fetchList(listCmd(), pagedFetcher(95, 30, &calls))
	if err != nil {
		t.Fatalf("fetchList: %v", err)
	}

	if len(items) != 95 {
		t.Errorf("got %d items, want all 95", len(items))
	}
	for i, v := range items {
		if v != i {
			t.Fatalf("item %d = %d, pages came back out of order or overlapped", i, v)
		}
	}
}

// The server clamps PageSize to MAX_RESPONSE_ITEMS. Comparing each page against
// the requested size would treat the very first (clamped) page as the last one.
func TestFetchListAllHandlesServerClampedPageSize(t *testing.T) {
	fetchAll = true
	defer func() { fetchAll = false }()

	cmd := listCmd()
	cmd.Flags().Set("per-page", "500")

	var calls []gitea.ListOptions
	items, err := fetchList(cmd, pagedFetcher(95, 30, &calls))
	if err != nil {
		t.Fatalf("fetchList: %v", err)
	}

	if len(items) != 95 {
		t.Fatalf("got %d items, want all 95 despite the server clamping page size", len(items))
	}
}

func TestFetchListAllStopsOnExactMultiple(t *testing.T) {
	fetchAll = true
	defer func() { fetchAll = false }()

	var calls []gitea.ListOptions
	items, err := fetchList(listCmd(), pagedFetcher(60, 30, &calls))
	if err != nil {
		t.Fatalf("fetchList: %v", err)
	}

	if len(items) != 60 {
		t.Errorf("got %d items, want 60", len(items))
	}
	// Two full pages plus the empty page that proves the list ended.
	if len(calls) != 3 {
		t.Errorf("made %d requests, want 3", len(calls))
	}
}

func TestListOptionsReadsFlags(t *testing.T) {
	cmd := listCmd()
	cmd.Flags().Set("page", "3")
	cmd.Flags().Set("per-page", "7")

	opts := listOptions(cmd)
	if opts.Page != 3 || opts.PageSize != 7 {
		t.Errorf("listOptions = %+v, want Page 3 / PageSize 7", opts)
	}
}

func TestListOptionsWithoutFlags(t *testing.T) {
	opts := listOptions(&cobra.Command{Use: "get"})
	if opts.Page != 0 || opts.PageSize != 0 {
		t.Errorf("listOptions = %+v, want zero values so the server default applies", opts)
	}
}
