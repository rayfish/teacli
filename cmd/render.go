package cmd

import (
	"fmt"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/output"

	"github.com/spf13/cobra"
)

// detail collects the key/value lines that make up the text rendering of a
// single resource. Commands that fetch one object use it instead of handing the
// struct to the generic printer, which would emit Go's %v syntax including raw
// pointer addresses.
type detail struct {
	rows  [][2]string
	width int
}

// add records a field, skipping it when the value is empty. Absent fields are
// noise in text output; --format json still carries them.
func (d *detail) add(key string, value any) {
	s := renderValue(value)
	if s == "" {
		return
	}
	d.set(key, s)
}

// always records a field even when its value is empty or a zero, for fields
// where the zero value is itself the answer (a false flag, a count of 0).
func (d *detail) always(key string, value any) {
	d.set(key, renderValue(value))
}

func (d *detail) set(key, value string) {
	if len(key) > d.width {
		d.width = len(key)
	}
	d.rows = append(d.rows, [2]string{key, value})
}

// print writes the collected fields as aligned "Key: value" lines. Values
// containing newlines are indented under their key so the block stays readable.
func (d *detail) print(p *output.Printer) error {
	for _, row := range d.rows {
		value := row[1]
		if strings.Contains(value, "\n") {
			p.Printf("%s:\n", row[0])
			for _, line := range strings.Split(strings.TrimRight(value, "\n"), "\n") {
				p.Printf("  %s\n", line)
			}
			continue
		}
		p.Printf("%-*s  %s\n", d.width+1, row[0]+":", value)
	}
	return nil
}

// renderValue turns a field into its text form. Times become RFC3339, nil
// pointers and zero times become empty so add skips them.
func renderValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimRight(v, "\n")
	case time.Time:
		if v.IsZero() {
			return ""
		}
		return v.Format(time.RFC3339)
	case *time.Time:
		if v == nil || v.IsZero() {
			return ""
		}
		return v.Format(time.RFC3339)
	case []string:
		return strings.Join(v, ", ")
	case bool:
		return fmt.Sprintf("%t", v)
	case gitea.StateType:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// userName renders a user reference for a detail line, tolerating a nil user.
func userName(u *gitea.User) string {
	if u == nil {
		return ""
	}
	return u.UserName
}

func labelNames(labels []*gitea.Label) []string {
	names := make([]string, 0, len(labels))
	for _, l := range labels {
		if l != nil {
			names = append(names, l.Name)
		}
	}
	return names
}

func userNames(users []*gitea.User) []string {
	names := make([]string, 0, len(users))
	for _, u := range users {
		if u != nil {
			names = append(names, u.UserName)
		}
	}
	return names
}

// cell prepares a value for a table cell: it collapses any embedded newlines
// (a multi-line commit or tag message would otherwise shear the table apart)
// and truncates on rune boundaries so multi-byte text is not cut in half.
func cell(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if max <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

// emitUserTable renders a list of users, used anywhere a command returns
// members, collaborators, reviewers, followers or assignees.
func emitUserTable(cmd *cobra.Command, users []*gitea.User, empty string) error {
	printer := getPrinter(cmd)
	return printer.Emit(users, func() error {
		if len(users) == 0 {
			printer.Println(empty)
			return nil
		}
		rows := make([][]string, 0, len(users))
		for _, u := range users {
			rows = append(rows, []string{
				fmt.Sprintf("%d", u.ID), u.UserName, u.FullName, u.Email, yesNo(u.IsAdmin),
			})
		}
		return printer.PrintTable([]string{"ID", "Login", "Name", "Email", "Admin"}, rows)
	})
}

func printUserText(printer *output.Printer, user *gitea.User) error {
	var d detail
	d.always("ID", user.ID)
	d.add("Login", user.UserName)
	d.add("Name", user.FullName)
	d.add("Email", user.Email)
	d.always("Admin", user.IsAdmin)
	d.always("Active", user.IsActive)
	if user.Restricted {
		d.always("Restricted", true)
	}
	d.add("Location", user.Location)
	d.add("Website", user.Website)
	d.add("Description", user.Description)
	d.add("Created", user.Created)
	d.add("Last login", user.LastLogin)
	return d.print(printer)
}

func printIssueText(printer *output.Printer, issue *gitea.Issue) error {
	var d detail
	d.always("Number", issue.Index)
	d.add("Title", issue.Title)
	d.always("State", issue.State)
	d.add("Author", userName(issue.Poster))
	d.add("Assignees", userNames(issue.Assignees))
	d.add("Labels", labelNames(issue.Labels))
	if issue.Milestone != nil {
		d.add("Milestone", issue.Milestone.Title)
	}
	if issue.IsLocked {
		d.always("Locked", true)
	}
	d.always("Comments", issue.Comments)
	d.add("Created", issue.Created)
	d.add("Updated", issue.Updated)
	d.add("Closed", issue.Closed)
	d.add("Due", issue.Deadline)
	d.add("URL", issue.HTMLURL)
	d.add("Body", issue.Body)
	return d.print(printer)
}

func printBranchText(printer *output.Printer, branch *gitea.Branch) error {
	var d detail
	d.add("Name", branch.Name)
	if branch.Commit != nil {
		d.add("Commit", branch.Commit.ID)
		d.add("Message", strings.TrimSpace(branch.Commit.Message))
		if branch.Commit.Author != nil {
			d.add("Author", branch.Commit.Author.Name)
		}
		d.add("Committed", branch.Commit.Timestamp)
	}
	d.always("Protected", branch.Protected)
	if branch.Protected {
		d.add("Protection", branch.EffectiveBranchProtectionName)
		d.always("Required approvals", branch.RequiredApprovals)
		d.always("Status check", branch.EnableStatusCheck)
		d.add("Status contexts", branch.StatusCheckContexts)
	}
	d.always("Can push", branch.UserCanPush)
	d.always("Can merge", branch.UserCanMerge)
	return d.print(printer)
}

func printReleaseText(printer *output.Printer, rel *gitea.Release) error {
	var d detail
	d.always("ID", rel.ID)
	d.add("Tag", rel.TagName)
	d.add("Title", rel.Title)
	d.add("Target", rel.Target)
	d.always("Draft", rel.IsDraft)
	d.always("Prerelease", rel.IsPrerelease)
	d.add("Author", userName(rel.Publisher))
	d.add("Created", rel.CreatedAt)
	d.add("Published", rel.PublishedAt)
	d.add("URL", rel.HTMLURL)
	if len(rel.Attachments) > 0 {
		names := make([]string, 0, len(rel.Attachments))
		for _, a := range rel.Attachments {
			if a != nil {
				names = append(names, fmt.Sprintf("%s (%d, id %d)", a.Name, a.Size, a.ID))
			}
		}
		d.add("Assets", names)
	}
	d.add("Notes", rel.Note)
	return d.print(printer)
}

func printOrgText(printer *output.Printer, org *gitea.Organization) error {
	var d detail
	d.always("ID", org.ID)
	d.add("Name", org.Name)
	d.add("Full name", org.FullName)
	d.add("Description", org.Description)
	d.add("Email", org.Email)
	d.add("Website", org.Website)
	d.add("Location", org.Location)
	d.add("Visibility", org.Visibility)
	d.always("Repo admins change team access", org.RepoAdminChangeTeamAccess)
	return d.print(printer)
}

func printMilestoneText(printer *output.Printer, ms *gitea.Milestone) error {
	var d detail
	d.always("ID", ms.ID)
	d.add("Title", ms.Title)
	d.always("State", ms.State)
	d.always("Open issues", ms.OpenIssues)
	d.always("Closed issues", ms.ClosedIssues)
	d.add("Due", ms.Deadline)
	d.add("Created", ms.Created)
	d.add("Updated", ms.Updated)
	d.add("Closed", ms.Closed)
	d.add("Description", ms.Description)
	return d.print(printer)
}

func printHookText(printer *output.Printer, hook *gitea.Hook) error {
	var d detail
	d.always("ID", hook.ID)
	d.add("Type", hook.Type)
	d.add("URL", hook.Config["url"])
	d.add("Content type", hook.Config["content_type"])
	d.always("Active", hook.Active)
	d.add("Events", hook.Events)
	d.add("Branch filter", hook.BranchFilter)
	d.add("Created", hook.Created)
	d.add("Updated", hook.Updated)
	return d.print(printer)
}

func printActionRunText(printer *output.Printer, run *gitea.ActionWorkflowRun) error {
	var d detail
	d.always("ID", run.ID)
	d.always("Run number", run.RunNumber)
	d.add("Title", run.DisplayTitle)
	d.add("Workflow", run.Path)
	d.always("Status", run.Status)
	d.add("Conclusion", run.Conclusion)
	d.add("Event", run.Event)
	d.add("Branch", run.HeadBranch)
	d.add("Commit", run.HeadSha)
	d.add("Actor", userName(run.Actor))
	d.add("Trigger actor", userName(run.TriggerActor))
	d.always("Attempt", run.RunAttempt)
	d.add("Started", run.StartedAt)
	d.add("Completed", run.CompletedAt)
	d.add("URL", run.HTMLURL)
	return d.print(printer)
}

func printActionJobText(printer *output.Printer, job *gitea.ActionWorkflowJob) error {
	var d detail
	d.always("ID", job.ID)
	d.add("Name", job.Name)
	d.always("Run ID", job.RunID)
	d.always("Status", job.Status)
	d.add("Conclusion", job.Conclusion)
	d.add("Branch", job.HeadBranch)
	d.add("Commit", job.HeadSha)
	d.add("Runner", job.RunnerName)
	d.add("Labels", job.Labels)
	d.add("Created", job.CreatedAt)
	d.add("Started", job.StartedAt)
	d.add("Completed", job.CompletedAt)
	d.add("URL", job.HTMLURL)
	if len(job.Steps) > 0 {
		lines := make([]string, 0, len(job.Steps))
		for _, s := range job.Steps {
			if s == nil {
				continue
			}
			status := s.Status
			if s.Conclusion != "" {
				status = s.Conclusion
			}
			lines = append(lines, fmt.Sprintf("%-10s %s", status, s.Name))
		}
		d.add("Steps", strings.Join(lines, "\n"))
	}
	return d.print(printer)
}
