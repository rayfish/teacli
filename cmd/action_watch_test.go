package cmd

import (
	stderrors "errors"
	"strings"
	"testing"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"
)

// fakeClock advances only when the watcher sleeps, so tests cover long waits
// without spending the wall-clock time.
type fakeClock struct {
	current time.Time
	slept   []time.Duration
}

func (c *fakeClock) now() time.Time { return c.current }

func (c *fakeClock) sleep(d time.Duration) {
	c.slept = append(c.slept, d)
	c.current = c.current.Add(d)
}

// runSequence hands back one run per call, repeating the last one forever.
func runSequence(runs ...*gitea.ActionWorkflowRun) func() (*gitea.ActionWorkflowRun, error) {
	call := 0
	return func() (*gitea.ActionWorkflowRun, error) {
		run := runs[call]
		if call < len(runs)-1 {
			call++
		}
		return run, nil
	}
}

func run(status, conclusion string) *gitea.ActionWorkflowRun {
	return &gitea.ActionWorkflowRun{ID: 42, RunNumber: 7, Status: status, Conclusion: conclusion}
}

func TestWatchRunReturnsTheRunOnceItFinishes(t *testing.T) {
	clock := &fakeClock{}
	watcher := runWatcher{
		fetch:    runSequence(run("queued", ""), run("in_progress", ""), run("completed", "success")),
		interval: 5 * time.Second,
		timeout:  time.Hour,
		now:      clock.now,
		sleep:    clock.sleep,
	}

	finished, err := watcher.wait()
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if finished.Conclusion != "success" {
		t.Errorf("conclusion = %q, want success", finished.Conclusion)
	}
	if len(clock.slept) != 2 {
		t.Errorf("slept %d times, want 2 (one per poll that found the run still running)", len(clock.slept))
	}
}

func TestWatchRunReturnsImmediatelyWhenTheRunIsAlreadyDone(t *testing.T) {
	clock := &fakeClock{}
	watcher := runWatcher{
		fetch:    runSequence(run("completed", "failure")),
		interval: 5 * time.Second,
		timeout:  time.Hour,
		now:      clock.now,
		sleep:    clock.sleep,
	}

	if _, err := watcher.wait(); err != nil {
		t.Fatalf("wait: %v", err)
	}
	if len(clock.slept) != 0 {
		t.Errorf("slept %d times, want 0: a finished run should not wait an interval", len(clock.slept))
	}
}

func TestWatchRunReportsEachStatusOnlyOnce(t *testing.T) {
	clock := &fakeClock{}
	var seen []string
	watcher := runWatcher{
		fetch: runSequence(
			run("queued", ""),
			run("in_progress", ""),
			run("in_progress", ""),
			run("completed", "success"),
		),
		interval: 5 * time.Second,
		timeout:  time.Hour,
		now:      clock.now,
		sleep:    clock.sleep,
		onChange: func(status string) { seen = append(seen, status) },
	}

	if _, err := watcher.wait(); err != nil {
		t.Fatalf("wait: %v", err)
	}

	want := []string{"queued", "in_progress", "completed"}
	if strings.Join(seen, ",") != strings.Join(want, ",") {
		t.Errorf("statuses = %v, want %v (the repeated in_progress poll should not report again)", seen, want)
	}
}

func TestWatchRunTimesOut(t *testing.T) {
	clock := &fakeClock{}
	watcher := runWatcher{
		fetch:    runSequence(run("in_progress", "")),
		interval: 10 * time.Second,
		timeout:  30 * time.Second,
		now:      clock.now,
		sleep:    clock.sleep,
	}

	_, err := watcher.wait()
	if err == nil {
		t.Fatal("wait returned no error, want a timeout")
	}
	if errors.ExitCode(err) != errors.ExitGeneralError {
		t.Errorf("exit code = %d, want %d", errors.ExitCode(err), errors.ExitGeneralError)
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %q, want it to mention the timeout", err.Error())
	}
	if clock.current.Sub(time.Time{}) > 30*time.Second {
		t.Errorf("waited %v, want to stop at the 30s timeout", clock.current.Sub(time.Time{}))
	}
}

func TestWatchRunWaitsForeverWithoutATimeout(t *testing.T) {
	clock := &fakeClock{}
	fetch := runSequence(
		run("in_progress", ""), run("in_progress", ""), run("in_progress", ""),
		run("completed", "success"),
	)
	watcher := runWatcher{
		fetch:    fetch,
		interval: time.Hour,
		timeout:  0,
		now:      clock.now,
		sleep:    clock.sleep,
	}

	finished, err := watcher.wait()
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if finished.Conclusion != "success" {
		t.Errorf("conclusion = %q, want success", finished.Conclusion)
	}
}

func TestWatchRunPropagatesAFetchError(t *testing.T) {
	wanted := errors.NewNotFoundError("workflow run", "42")
	watcher := runWatcher{
		fetch:    func() (*gitea.ActionWorkflowRun, error) { return nil, wanted },
		interval: time.Second,
		timeout:  time.Hour,
		now:      func() time.Time { return time.Time{} },
		sleep:    func(time.Duration) {},
	}

	_, err := watcher.wait()
	if !stderrors.Is(err, error(wanted)) {
		t.Errorf("error = %v, want the fetch error unchanged", err)
	}
}

func TestRunInFlightStatuses(t *testing.T) {
	tests := []struct {
		status   string
		inFlight bool
	}{
		{"pending", true},
		{"queued", true},
		{"waiting", true},
		{"running", true},
		{"in_progress", true},
		{"blocked", true},
		{"completed", false},
		{"success", false},
		{"failure", false},
		{"cancelled", false},
		{"skipped", false},
	}
	for _, tt := range tests {
		if got := runInFlight(tt.status); got != tt.inFlight {
			t.Errorf("runInFlight(%q) = %v, want %v", tt.status, got, tt.inFlight)
		}
	}
}

func TestRunOutcomePrefersTheConclusion(t *testing.T) {
	if got := runOutcome(run("completed", "failure")); got != "failure" {
		t.Errorf("runOutcome = %q, want failure", got)
	}
	// Gitea leaves conclusion empty and puts the result in status on some runs.
	if got := runOutcome(run("success", "")); got != "success" {
		t.Errorf("runOutcome = %q, want success", got)
	}
}

func TestWatchOutcomeErrorNamesTheFailedRun(t *testing.T) {
	err := watchOutcomeError(run("completed", "failure"), "16338")
	if err == nil {
		t.Fatal("watchOutcomeError returned nil for a failed run")
	}
	if errors.ExitCode(err) != errors.ExitGeneralError {
		t.Errorf("exit code = %d, want %d", errors.ExitCode(err), errors.ExitGeneralError)
	}
	if !strings.Contains(err.Error(), "16338") || !strings.Contains(err.Error(), "failure") {
		t.Errorf("error = %q, want it to name the run and the conclusion", err.Error())
	}
}

func TestWatchOutcomeErrorIsNilForASuccessfulRun(t *testing.T) {
	if err := watchOutcomeError(run("completed", "success"), "16338"); err != nil {
		t.Errorf("watchOutcomeError = %v, want nil for a successful run", err)
	}
}

func TestWatchRejectsANonPositiveInterval(t *testing.T) {
	if err := validateWatchTiming(0, time.Minute); err == nil {
		t.Error("validateWatchTiming accepted a zero interval, want a validation error")
	} else if errors.ExitCode(err) != errors.ExitValidationError {
		t.Errorf("exit code = %d, want %d", errors.ExitCode(err), errors.ExitValidationError)
	}
	if err := validateWatchTiming(time.Second, -time.Minute); err == nil {
		t.Error("validateWatchTiming accepted a negative timeout, want a validation error")
	}
	if err := validateWatchTiming(time.Second, 0); err != nil {
		t.Errorf("validateWatchTiming(1s, 0) = %v, want nil: zero means no timeout", err)
	}
}

func TestActionWatchIsRegisteredWithItsWaitAlias(t *testing.T) {
	watch := findCmd(RootCmd, "action", "watch")
	if watch == nil {
		t.Fatal("action watch command not found")
	}
	if wait := findCmd(RootCmd, "action", "wait"); wait != watch {
		t.Error("action wait does not resolve to the watch command")
	}
	if err := watch.Args(watch, []string{"owner/repo", "42"}); err != nil {
		t.Errorf("watch rejected an explicit repo and run: %v", err)
	}
	if err := watch.Args(watch, []string{"42"}); err != nil {
		t.Errorf("watch rejected an inferred repo: %v", err)
	}
	if err := watch.Args(watch, nil); err == nil {
		t.Error("watch accepted no arguments, want a run to be required")
	}
}

func TestActionWatchFlagDefaults(t *testing.T) {
	watch := findCmd(RootCmd, "action", "watch")
	if watch == nil {
		t.Fatal("action watch command not found")
	}
	if got := watch.Flag("interval").DefValue; got != "5s" {
		t.Errorf("--interval default = %q, want 5s", got)
	}
	if got := watch.Flag("timeout").DefValue; got != "30m0s" {
		t.Errorf("--timeout default = %q, want 30m0s", got)
	}
}

func TestWatchRunPollsOnceMoreAtTheDeadline(t *testing.T) {
	clock := &fakeClock{}
	polls := 0
	watcher := runWatcher{
		fetch: func() (*gitea.ActionWorkflowRun, error) {
			polls++
			// The run lands right on the deadline, which the last poll must see.
			if clock.current.Sub(time.Time{}) >= 25*time.Second {
				return run("completed", "success"), nil
			}
			return run("in_progress", ""), nil
		},
		interval: 10 * time.Second,
		timeout:  25 * time.Second,
		now:      clock.now,
		sleep:    clock.sleep,
	}

	finished, err := watcher.wait()
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if finished.Conclusion != "success" {
		t.Errorf("conclusion = %q, want success", finished.Conclusion)
	}
	if polls != 4 {
		t.Errorf("polled %d times, want 4: at 0s, 10s, 20s and once more at the 25s deadline", polls)
	}
	want := []time.Duration{10 * time.Second, 10 * time.Second, 5 * time.Second}
	if len(clock.slept) != len(want) {
		t.Fatalf("slept %v, want %v (the last wait is trimmed to the deadline)", clock.slept, want)
	}
	for i, d := range want {
		if clock.slept[i] != d {
			t.Errorf("sleep %d = %v, want %v", i, clock.slept[i], d)
		}
	}
}
