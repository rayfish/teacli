package cmd

import (
	"strings"
	"testing"
	"time"

	"code.gitea.io/sdk/gitea"
)

func TestCellCollapsesNewlines(t *testing.T) {
	// A multi-line commit or tag message would otherwise shear a table apart.
	got := cell("first line\n\nsecond line", 0)
	if got != "first line second line" {
		t.Errorf("cell = %q, want the lines joined by a space", got)
	}
}

func TestCellTruncatesOnRuneBoundaries(t *testing.T) {
	got := cell("ünïcödé text that runs on", 10)
	if len([]rune(got)) != 10 {
		t.Errorf("cell = %q (%d runes), want 10 runes", got, len([]rune(got)))
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("cell = %q, want a truncation marker", got)
	}
	// Cutting mid-rune would produce replacement characters.
	if strings.ContainsRune(got, '�') {
		t.Errorf("cell = %q, cut through a multi-byte rune", got)
	}
}

func TestCellLeavesShortValuesAlone(t *testing.T) {
	if got := cell("short", 40); got != "short" {
		t.Errorf("cell = %q, want %q", got, "short")
	}
}

func TestRenderValueSkipsZeroTimes(t *testing.T) {
	if got := renderValue(time.Time{}); got != "" {
		t.Errorf("renderValue(zero time) = %q, want empty so detail.add skips it", got)
	}
	var nilTime *time.Time
	if got := renderValue(nilTime); got != "" {
		t.Errorf("renderValue(nil *time.Time) = %q, want empty", got)
	}
	when := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	if got := renderValue(when); got != "2026-08-16T12:00:00Z" {
		t.Errorf("renderValue = %q, want RFC3339", got)
	}
}

func TestDetailAddSkipsEmptyButAlwaysKeeps(t *testing.T) {
	var d detail
	d.add("Skipped", "")
	d.always("Kept", false)
	if len(d.rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(d.rows))
	}
	if d.rows[0][0] != "Kept" || d.rows[0][1] != "false" {
		t.Errorf("row = %v, want [Kept false]", d.rows[0])
	}
}

func TestUserNameToleratesNil(t *testing.T) {
	if got := userName(nil); got != "" {
		t.Errorf("userName(nil) = %q, want empty", got)
	}
	if got := userName(&gitea.User{UserName: "alice"}); got != "alice" {
		t.Errorf("userName = %q, want alice", got)
	}
}

func TestShortSHA(t *testing.T) {
	if got := shortSHA("228517443c6e75efabffa39cd67ef5b2b774b3de"); got != "228517443c" {
		t.Errorf("shortSHA = %q, want 228517443c", got)
	}
	if got := shortSHA("abc"); got != "abc" {
		t.Errorf("shortSHA on a short value = %q, want it unchanged", got)
	}
}
