package cmd

import (
	"strings"
	"testing"

	"github.com/rayfish/teacli/modules/errors"
)

func TestAPIErrorUsesGiteaMessage(t *testing.T) {
	err := apiError(404, []byte(`{"message":"pull request not found"}`))

	if err.Code != errors.ExitNotFound {
		t.Errorf("code = %d, want %d", err.Code, errors.ExitNotFound)
	}
	if err.Message != "pull request not found" {
		t.Errorf("message = %q, want the server's message", err.Message)
	}
}

func TestAPIErrorMapsStatusToExitCode(t *testing.T) {
	for status, want := range map[int]int{
		401: errors.ExitPermissionDenied,
		403: errors.ExitPermissionDenied,
		404: errors.ExitNotFound,
		400: errors.ExitValidationError,
		422: errors.ExitValidationError,
		500: errors.ExitGeneralError,
	} {
		if got := apiError(status, nil).Code; got != want {
			t.Errorf("status %d mapped to exit %d, want %d", status, got, want)
		}
	}
}

func TestAPIErrorDropsHTMLBodies(t *testing.T) {
	page := []byte("<!DOCTYPE HTML>\n<html><body><h1>Error response</h1></body></html>")

	err := apiError(404, page)
	if strings.Contains(err.Message, "<") {
		t.Errorf("message = %q, want the HTML page dropped rather than echoed", err.Message)
	}
	if err.Message != "Not Found" {
		t.Errorf("message = %q, want the status text as the fallback", err.Message)
	}
}

func TestAPIErrorTruncatesLongBodies(t *testing.T) {
	err := apiError(500, []byte(strings.Repeat("x", 5000)))

	if len(err.Message) > maxErrorBody+3 {
		t.Errorf("message is %d chars, want it truncated to about %d", len(err.Message), maxErrorBody)
	}
}

func TestAPIErrorFallsBackToStatusText(t *testing.T) {
	if got := apiError(403, []byte("   ")).Message; got != "Forbidden" {
		t.Errorf("message = %q, want \"Forbidden\"", got)
	}
}
