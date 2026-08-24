package output

import (
	"encoding/json"
	"os"
	"testing"
)

// captureStdout runs fn and returns whatever it wrote to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = orig
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

func TestEmitJSONPrintsData(t *testing.T) {
	p := NewPrinter(FormatJSON)
	out := captureStdout(t, func() {
		_ = p.Emit(map[string]int{"number": 7}, func() error {
			t.Fatal("textFn should not run for json format")
			return nil
		})
	})
	var got map[string]int
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not valid JSON: %q", out)
	}
	if got["number"] != 7 {
		t.Errorf("got %v, want number=7", got)
	}
}

func TestEmitTextRunsTextFn(t *testing.T) {
	p := NewPrinter(FormatText)
	ran := false
	_ = p.Emit(nil, func() error { ran = true; return nil })
	if !ran {
		t.Error("textFn did not run for text format")
	}
}
