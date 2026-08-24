// Package errors defines custom error types and exit codes for teacli.
package errors

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"code.gitea.io/sdk/gitea"
)

// Exit codes
const (
	ExitSuccess          = 0
	ExitGeneralError     = 1
	ExitValidationError  = 2
	ExitNotFound         = 3
	ExitPermissionDenied = 4
)

// CLIError represents a structured error with exit code
type CLIError struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e *CLIError) Error() string {
	return e.Message
}

// NewValidationError creates a validation error (exit code 2)
func NewValidationError(message string, details map[string]interface{}) *CLIError {
	return &CLIError{
		Code:    ExitValidationError,
		Message: message,
		Details: details,
	}
}

// NewNotFoundError creates a not found error (exit code 3)
func NewNotFoundError(resourceType, id string) *CLIError {
	return &CLIError{
		Code:    ExitNotFound,
		Message: fmt.Sprintf("%s not found: %s", resourceType, id),
	}
}

// NewPermissionError creates a permission denied error (exit code 4)
func NewPermissionError(message string) *CLIError {
	return &CLIError{
		Code:    ExitPermissionDenied,
		Message: "permission denied: " + message,
	}
}

// NewGeneralError creates a general error (exit code 1)
func NewGeneralError(message string) *CLIError {
	return &CLIError{
		Code:    ExitGeneralError,
		Message: message,
	}
}

// Message renders an error for a human reader: the message, followed by any
// details the error carries, which is usually where the useful hint lives.
func Message(err error) string {
	if err == nil {
		return ""
	}

	var cliErr *CLIError
	if !errors.As(err, &cliErr) {
		return err.Error()
	}
	if len(cliErr.Details) == 0 {
		return cliErr.Message
	}

	keys := make([]string, 0, len(cliErr.Details))
	for k := range cliErr.Details {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %s", k, formatDetail(cliErr.Details[k])))
	}

	return fmt.Sprintf("%s (%s)", cliErr.Message, strings.Join(parts, ", "))
}

// formatDetail renders a detail value, flattening string lists so they read as
// prose rather than Go slice syntax.
func formatDetail(value interface{}) string {
	switch v := value.(type) {
	case []string:
		return strings.Join(v, ", ")
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ExitCode returns the exit code an error should terminate with.
func ExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		return cliErr.Code
	}
	return ExitGeneralError
}

// FromGiteaNotFound behaves like FromGitea, but on a 404 it substitutes a typed
// "<resourceType> not found: <id>" message. Gitea returns a generic "not found"
// body for missing resources, so callers that fetch a single resource by ID should
// use this to keep errors specific and greppable.
func FromGiteaNotFound(resp *gitea.Response, err error, resourceType, id string) *CLIError {
	if err == nil {
		return nil
	}
	if resp != nil && resp.Response != nil && resp.StatusCode == http.StatusNotFound {
		return NewNotFoundError(resourceType, id)
	}
	return FromGitea(resp, err)
}

// FromGitea maps a Gitea SDK response and error to a CLIError with the right exit
// code. The SDK collapses HTTP errors into plain errors, so the status code is read
// from resp, not from the error text.
func FromGitea(resp *gitea.Response, err error) *CLIError {
	if err == nil {
		return nil
	}
	if resp == nil || resp.Response == nil {
		return NewGeneralError(err.Error())
	}
	switch resp.StatusCode {
	case http.StatusNotFound:
		return &CLIError{Code: ExitNotFound, Message: err.Error()}
	case http.StatusForbidden, http.StatusUnauthorized:
		return &CLIError{Code: ExitPermissionDenied, Message: err.Error()}
	case http.StatusUnprocessableEntity, http.StatusBadRequest:
		return &CLIError{Code: ExitValidationError, Message: err.Error()}
	default:
		return NewGeneralError(err.Error())
	}
}
