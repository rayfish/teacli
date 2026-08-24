// Package cmd contains notification commands.
package cmd

import (
	"fmt"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var notificationCmd = &cobra.Command{
	Use:     "notification",
	Aliases: []string{"notifications", "notif"},
	Short:   "Read and manage notifications",
}

var notificationListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List notification threads, across all repositories or one repository",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runNotificationList,
}

var notificationGetCmd = &cobra.Command{
	Use:   "get <thread-id>",
	Short: "Get one notification thread",
	Args:  cobra.ExactArgs(1),
	RunE:  runNotificationGet,
}

var notificationCountCmd = &cobra.Command{
	Use:     "count",
	Aliases: []string{"check"},
	Short:   "Report how many unread notifications you have",
	Args:    cobra.NoArgs,
	RunE:    runNotificationCount,
}

var notificationReadCmd = &cobra.Command{
	Use:   "read [<thread-id>]",
	Short: "Mark one thread, or every unread thread, as read",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runNotificationRead,
}

var notificationUnreadCmd = &cobra.Command{
	Use:   "unread <thread-id>",
	Short: "Mark a thread as unread",
	Args:  cobra.ExactArgs(1),
	RunE:  runNotificationUnread,
}

var notificationPinCmd = &cobra.Command{
	Use:   "pin <thread-id>",
	Short: "Pin a notification thread",
	Args:  cobra.ExactArgs(1),
	RunE:  runNotificationPin,
}

func init() {
	RootCmd.AddCommand(notificationCmd)
	notificationCmd.AddCommand(notificationListCmd, notificationGetCmd, notificationCountCmd,
		notificationReadCmd, notificationUnreadCmd, notificationPinCmd)

	addPageFlags(notificationListCmd)
	notificationListCmd.Flags().StringSliceP("status", "s", []string{"unread"},
		"Statuses to include: unread, read, pinned")
	notificationListCmd.Flags().StringSlice("type", nil,
		"Subject types to include: issue, pull, commit, repository")
	notificationListCmd.Flags().String("since", "", "Only threads updated at or after this RFC3339 time")
	notificationListCmd.Flags().String("before", "", "Only threads updated before this RFC3339 time")

	// No local --all here: the global --all means auto-paginate, and a local
	// flag of the same name would shadow it. Omitting the thread id already
	// means "every unread thread".
	notificationReadCmd.Flags().String("repo", "", "Limit a bulk read to this owner/repo")
}

// parseNotifyStatuses validates the --status values, which the API rejects
// silently rather than with an error when they are wrong.
func parseNotifyStatuses(values []string) ([]gitea.NotifyStatus, error) {
	var out []gitea.NotifyStatus
	for _, v := range values {
		switch strings.ToLower(v) {
		case "unread":
			out = append(out, gitea.NotifyStatusUnread)
		case "read":
			out = append(out, gitea.NotifyStatusRead)
		case "pinned":
			out = append(out, gitea.NotifyStatusPinned)
		default:
			return nil, errors.NewValidationError(fmt.Sprintf("invalid status: %s", v),
				map[string]interface{}{"expected": "unread, read, or pinned"})
		}
	}
	return out, nil
}

func parseSubjectTypes(values []string) ([]gitea.NotifySubjectType, error) {
	var out []gitea.NotifySubjectType
	for _, v := range values {
		switch strings.ToLower(v) {
		case "issue":
			out = append(out, gitea.NotifySubjectIssue)
		case "pull", "pr":
			out = append(out, gitea.NotifySubjectPull)
		case "commit":
			out = append(out, gitea.NotifySubjectCommit)
		case "repository", "repo":
			out = append(out, gitea.NotifySubjectRepository)
		default:
			return nil, errors.NewValidationError(fmt.Sprintf("invalid subject type: %s", v),
				map[string]interface{}{"expected": "issue, pull, commit, or repository"})
		}
	}
	return out, nil
}

// timeFlag parses an optional RFC3339 flag into a time, leaving it zero when
// the flag was not given.
func timeFlag(cmd *cobra.Command, name string) (time.Time, error) {
	raw, _ := cmd.Flags().GetString(name)
	if raw == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, errors.NewValidationError(fmt.Sprintf("invalid --%s: %s", name, raw),
			map[string]interface{}{"expected": "an RFC3339 timestamp, e.g. 2026-01-31T00:00:00Z"})
	}
	return t, nil
}

func runNotificationList(cmd *cobra.Command, args []string) error {
	statuses, _ := cmd.Flags().GetStringSlice("status")
	status, err := parseNotifyStatuses(statuses)
	if err != nil {
		return err
	}
	types, _ := cmd.Flags().GetStringSlice("type")
	subjectTypes, err := parseSubjectTypes(types)
	if err != nil {
		return err
	}
	since, err := timeFlag(cmd, "since")
	if err != nil {
		return err
	}
	before, err := timeFlag(cmd, "before")
	if err != nil {
		return err
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	var owner, repo string
	if len(args) == 1 {
		if owner, repo, err = repoArg(args[0]); err != nil {
			return err
		}
	}

	threads, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.NotificationThread, *gitea.Response, error) {
		opts := gitea.ListNotificationOptions{
			ListOptions:  lo,
			Status:       status,
			SubjectTypes: subjectTypes,
			Since:        since,
			Before:       before,
		}
		if owner != "" {
			return client.ListRepoNotifications(owner, repo, opts)
		}
		return client.ListNotifications(opts)
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(threads, func() error {
		if len(threads) == 0 {
			printer.Println("No notifications found.")
			return nil
		}
		rows := make([][]string, 0, len(threads))
		for _, t := range threads {
			repoName, title, kind, state := "", "", "", ""
			if t.Repository != nil {
				repoName = t.Repository.FullName
			}
			if t.Subject != nil {
				title = cell(t.Subject.Title, 50)
				kind = string(t.Subject.Type)
				state = string(t.Subject.State)
			}
			rows = append(rows, []string{
				fmt.Sprintf("%d", t.ID), repoName, kind, state, title, yesNo(t.Unread),
			})
		}
		return printer.PrintTable([]string{"ID", "Repo", "Type", "State", "Subject", "Unread"}, rows)
	})
}

func runNotificationGet(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "thread id")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	thread, resp, err := client.GetNotification(id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "notification thread", args[0])
	}

	printer := getPrinter(cmd)
	return printer.Emit(thread, func() error {
		var d detail
		d.always("ID", thread.ID)
		if thread.Repository != nil {
			d.add("Repository", thread.Repository.FullName)
		}
		if thread.Subject != nil {
			d.add("Subject", thread.Subject.Title)
			d.add("Type", string(thread.Subject.Type))
			d.add("State", string(thread.Subject.State))
			d.add("URL", thread.Subject.HTMLURL)
		}
		d.always("Unread", thread.Unread)
		d.always("Pinned", thread.Pinned)
		d.add("Updated", thread.UpdatedAt)
		return d.print(printer)
	})
}

func runNotificationCount(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	count, resp, err := client.CheckNotifications()
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]int64{"unread": count}, func() error {
		printer.Printf("%d unread\n", count)
		return nil
	})
}

func runNotificationRead(cmd *cobra.Command, args []string) error {
	return setNotificationStatus(cmd, args, gitea.NotifyStatusRead, "read")
}

func runNotificationUnread(cmd *cobra.Command, args []string) error {
	return setNotificationStatus(cmd, args, gitea.NotifyStatusUnread, "unread")
}

func runNotificationPin(cmd *cobra.Command, args []string) error {
	return setNotificationStatus(cmd, args, gitea.NotifyStatusPinned, "pinned")
}

// setNotificationStatus moves one thread, or every currently-unread thread, to
// the given status. Only `read` supports the bulk form, because that is the only
// bulk endpoint Gitea offers.
func setNotificationStatus(cmd *cobra.Command, args []string, to gitea.NotifyStatus, label string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	if len(args) == 1 {
		id, err := int64Arg(args[0], "thread id")
		if err != nil {
			return err
		}
		if dryRunf(cmd, "Would mark thread %d as %s", id, label) {
			return nil
		}
		thread, resp, err := client.ReadNotification(id, to)
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "notification thread", args[0])
		}
		return emitMessage(cmd, thread, "Marked thread %d as %s", id, label)
	}

	if to != gitea.NotifyStatusRead {
		return errors.NewValidationError(fmt.Sprintf("a thread id is required to mark a thread %s", label),
			map[string]interface{}{"hint": "only 'notification read' can act on every thread at once"})
	}

	repoFlag, _ := cmd.Flags().GetString("repo")
	opts := gitea.MarkNotificationOptions{
		Status:   []gitea.NotifyStatus{gitea.NotifyStatusUnread},
		ToStatus: gitea.NotifyStatusRead,
	}

	scope := "all repositories"
	if repoFlag != "" {
		scope = repoFlag
	}
	if dryRunf(cmd, "Would mark every unread thread in %s as read", scope) {
		return nil
	}

	var (
		threads []*gitea.NotificationThread
		resp    *gitea.Response
	)
	if repoFlag != "" {
		owner, repo, err := repoArg(repoFlag)
		if err != nil {
			return err
		}
		threads, resp, err = client.ReadRepoNotifications(owner, repo, opts)
		if err != nil {
			return errors.FromGitea(resp, err)
		}
	} else {
		threads, resp, err = client.ReadNotifications(opts)
		if err != nil {
			return errors.FromGitea(resp, err)
		}
	}

	return emitMessage(cmd, threads, "Marked %d thread(s) in %s as read", len(threads), scope)
}
