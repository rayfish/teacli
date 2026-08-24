package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

// apiDo sends an authenticated request to the Gitea REST API and returns the raw
// response. path is relative to /api/v1 unless it already names an absolute URL
// or starts with /api/.
func apiDo(cmd *cobra.Command, method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	server, err := getServer(cmd)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, apiURL(server.URL, path), body)
	if err != nil {
		return nil, errors.NewGeneralError(fmt.Sprintf("failed to build request: %v", err))
	}
	req.Header.Set("Authorization", "token "+server.Token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.NewGeneralError(err.Error())
	}
	return resp, nil
}

// apiURL joins a server base URL and an endpoint. Absolute URLs and paths that
// already carry an /api/ prefix are passed through untouched, so callers can
// reach non-v1 routes when they need to.
func apiURL(base, path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	base = strings.TrimSuffix(base, "/")
	if strings.HasPrefix(path, "/api/") {
		return base + path
	}
	return base + "/api/v1/" + strings.TrimPrefix(path, "/")
}

// apiRequest sends a request and returns the response body, mapping HTTP errors
// onto the CLI's exit codes. It exists for endpoints the SDK does not wrap;
// prefer the SDK client everywhere else.
func apiRequest(cmd *cobra.Command, method, path string, body any) ([]byte, error) {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, errors.NewGeneralError(fmt.Sprintf("failed to encode request: %v", err))
		}
		payload = bytes.NewReader(encoded)
	}

	resp, err := apiDo(cmd, method, path, payload, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewGeneralError(fmt.Sprintf("failed to read response: %v", err))
	}

	if resp.StatusCode >= 400 {
		return nil, apiError(resp.StatusCode, data)
	}

	return data, nil
}

// apiGetInto sends a GET and decodes the JSON body into out.
func apiGetInto(cmd *cobra.Command, path string, out any) error {
	data, err := apiRequest(cmd, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to decode response: %v", err))
	}
	return nil
}

// maxErrorBody bounds how much of an unrecognised error body is echoed back.
const maxErrorBody = 200

// apiError maps an HTTP status onto the CLI's exit codes, using the server's
// error message when it sends one.
func apiError(status int, body []byte) *errors.CLIError {
	message := ""

	var parsed struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Message != "" {
		message = parsed.Message
	} else {
		// Not a Gitea JSON error: an HTML error page from a proxy or a wrong
		// host. Echoing the whole page helps nobody, so keep a short excerpt.
		text := strings.TrimSpace(string(body))
		if !strings.HasPrefix(text, "<") {
			message = strings.Join(strings.Fields(text), " ")
			if len(message) > maxErrorBody {
				message = message[:maxErrorBody] + "..."
			}
		}
	}

	if message == "" {
		message = http.StatusText(status)
	}

	switch status {
	case http.StatusNotFound:
		return &errors.CLIError{Code: errors.ExitNotFound, Message: message}
	case http.StatusForbidden, http.StatusUnauthorized:
		return &errors.CLIError{Code: errors.ExitPermissionDenied, Message: message}
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return &errors.CLIError{Code: errors.ExitValidationError, Message: message}
	default:
		return errors.NewGeneralError(message)
	}
}

var apiCmd = &cobra.Command{
	Use:   "api <endpoint>",
	Short: "Make an authenticated request to any Gitea API endpoint",
	Long: `Send an authenticated HTTP request to the Gitea API and print the response body.

The endpoint is relative to /api/v1 unless it starts with /api/ or with a scheme,
so "repos/owner/name" and "/api/v1/repos/owner/name" are equivalent.

Build a JSON body with --field (always a string) and --raw-field (parsed as JSON,
so numbers, booleans, null, arrays and objects keep their type), or send one
verbatim with --input. --input accepts a file path, or - to read stdin. Supplying
a body switches the default method from GET to POST.

Use this for endpoints teacli has no typed command for. Typed commands give you
table output, validation and dry-run, so prefer them where they exist.`,
	Example: `  teacli api repos/owner/repo
  teacli api user/repos --paginate
  teacli api repos/owner/repo/issues --field title="bug" --field body="details"
  teacli api repos/owner/repo/issues/1 -X PATCH --raw-field state='"closed"'
  teacli api repos/owner/repo/releases --input release.json
  teacli api repos/owner/repo/issues -X POST --input - < issue.json`,
	Args: cobra.ExactArgs(1),
	RunE: runAPI,
}

func init() {
	RootCmd.AddCommand(apiCmd)

	apiCmd.Flags().StringP("method", "X", "", "HTTP method (default GET, or POST when a body is given)")
	apiCmd.Flags().StringArrayP("field", "f", nil, "Add a string field to the JSON body (key=value), repeatable")
	apiCmd.Flags().StringArrayP("raw-field", "F", nil, "Add a JSON-typed field to the body (key=value), repeatable")
	apiCmd.Flags().StringArrayP("header", "H", nil, "Add a request header (key:value), repeatable")
	apiCmd.Flags().StringP("input", "i", "", "Send this file as the request body, or - for stdin")
	apiCmd.Flags().Bool("include", false, "Write the status line and response headers to stderr")
	apiCmd.Flags().String("out", "", "Write the response body to this file instead of stdout")
	apiCmd.Flags().Bool("paginate", false, "Follow pagination and emit one merged JSON array")
	apiCmd.Flags().Int("per-page", 0, "Items per page when paginating")
}

func runAPI(cmd *cobra.Command, args []string) error {
	endpoint := args[0]

	method, _ := cmd.Flags().GetString("method")
	fields, _ := cmd.Flags().GetStringArray("field")
	rawFields, _ := cmd.Flags().GetStringArray("raw-field")
	headerArgs, _ := cmd.Flags().GetStringArray("header")
	input, _ := cmd.Flags().GetString("input")
	include, _ := cmd.Flags().GetBool("include")
	outPath, _ := cmd.Flags().GetString("out")
	paginate, _ := cmd.Flags().GetBool("paginate")

	if input != "" && (len(fields) > 0 || len(rawFields) > 0) {
		return errors.NewValidationError("--input cannot be combined with --field or --raw-field",
			map[string]interface{}{"hint": "build the body one way or the other"})
	}

	headers, err := parseHeaders(headerArgs)
	if err != nil {
		return err
	}

	body, err := apiBody(input, fields, rawFields)
	if err != nil {
		return err
	}

	method = strings.ToUpper(method)
	if method == "" {
		if body != nil {
			method = http.MethodPost
		} else {
			method = http.MethodGet
		}
	}

	if paginate && method != http.MethodGet {
		return errors.NewValidationError("--paginate only applies to GET requests", nil)
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun && method != http.MethodGet {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: %s %s\n", method, endpoint)
		if body != nil {
			printer.Printf("  body: %s\n", string(body))
		}
		return nil
	}

	out := io.Writer(os.Stdout)
	if outPath != "" {
		file, err := os.Create(outPath)
		if err != nil {
			return errors.NewGeneralError(fmt.Sprintf("failed to create %s: %v", outPath, err))
		}
		defer file.Close()
		out = file
	}

	if paginate {
		perPage, _ := cmd.Flags().GetInt("per-page")
		merged, err := apiPaginate(cmd, endpoint, headers, perPage)
		if err != nil {
			return err
		}
		_, err = out.Write(append(merged, '\n'))
		return err
	}

	var payload io.Reader
	if body != nil {
		payload = bytes.NewReader(body)
	}

	resp, err := apiDo(cmd, method, endpoint, payload, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if include {
		fmt.Fprintf(os.Stderr, "%s %s\n", resp.Proto, resp.Status)
		if err := resp.Header.Write(os.Stderr); err != nil {
			return errors.NewGeneralError(err.Error())
		}
		fmt.Fprintln(os.Stderr)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to read response: %v", err))
	}
	if resp.StatusCode >= 400 {
		return apiError(resp.StatusCode, data)
	}

	if len(data) == 0 {
		return nil
	}
	if _, err := out.Write(data); err != nil {
		return errors.NewGeneralError(err.Error())
	}
	if data[len(data)-1] != '\n' {
		fmt.Fprintln(out)
	}
	return nil
}

// apiPaginate walks every page of a list endpoint and merges the pages into a
// single JSON array. It stops on the first page shorter than the first one,
// matching how fetchList detects the end of a list.
func apiPaginate(cmd *cobra.Command, endpoint string, headers map[string]string, perPage int) ([]byte, error) {
	if perPage <= 0 {
		perPage = defaultPageSize
	}

	var merged []json.RawMessage
	effective := 0

	for page := 1; page <= maxPages; page++ {
		paged, err := withPageQuery(endpoint, page, perPage)
		if err != nil {
			return nil, err
		}

		resp, err := apiDo(cmd, http.MethodGet, paged, nil, headers)
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, errors.NewGeneralError(fmt.Sprintf("failed to read response: %v", readErr))
		}
		if resp.StatusCode >= 400 {
			return nil, apiError(resp.StatusCode, data)
		}

		var items []json.RawMessage
		if err := json.Unmarshal(data, &items); err != nil {
			return nil, errors.NewValidationError("--paginate needs an endpoint that returns a JSON array",
				map[string]interface{}{"endpoint": endpoint})
		}

		merged = append(merged, items...)

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

	if merged == nil {
		merged = []json.RawMessage{}
	}
	return json.MarshalIndent(merged, "", "  ")
}

// withPageQuery adds page and limit parameters to an endpoint, preserving any
// query string the caller already wrote.
func withPageQuery(endpoint string, page, perPage int) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", errors.NewValidationError(fmt.Sprintf("invalid endpoint: %v", err), nil)
	}
	q := parsed.Query()
	q.Set("page", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(perPage))
	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}

// apiBody builds the request body from --input or from the --field/--raw-field
// pairs. It returns nil when the caller asked for no body.
func apiBody(input string, fields, rawFields []string) ([]byte, error) {
	if input != "" {
		if input == "-" {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return nil, errors.NewGeneralError(fmt.Sprintf("failed to read stdin: %v", err))
			}
			return data, nil
		}
		data, err := os.ReadFile(input)
		if err != nil {
			return nil, errors.NewValidationError(fmt.Sprintf("failed to read %s: %v", input, err), nil)
		}
		return data, nil
	}

	if len(fields) == 0 && len(rawFields) == 0 {
		return nil, nil
	}

	obj := map[string]any{}
	for _, f := range fields {
		key, value, err := splitPair(f, "=", "--field")
		if err != nil {
			return nil, err
		}
		obj[key] = value
	}
	for _, f := range rawFields {
		key, value, err := splitPair(f, "=", "--raw-field")
		if err != nil {
			return nil, err
		}
		var decoded any
		if err := json.Unmarshal([]byte(value), &decoded); err != nil {
			return nil, errors.NewValidationError(fmt.Sprintf("--raw-field %s is not valid JSON: %s", key, value),
				map[string]interface{}{"hint": `quote strings as JSON, e.g. --raw-field state='"closed"'`})
		}
		obj[key] = decoded
	}

	return json.Marshal(obj)
}

func parseHeaders(args []string) (map[string]string, error) {
	if len(args) == 0 {
		return nil, nil
	}
	headers := make(map[string]string, len(args))
	for _, h := range args {
		key, value, err := splitPair(h, ":", "--header")
		if err != nil {
			return nil, err
		}
		headers[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return headers, nil
}

func splitPair(s, sep, flag string) (string, string, error) {
	key, value, found := strings.Cut(s, sep)
	if !found || key == "" {
		return "", "", errors.NewValidationError(fmt.Sprintf("invalid %s value: %q", flag, s),
			map[string]interface{}{"expected": "key" + sep + "value"})
	}
	return key, value, nil
}
