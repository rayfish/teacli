// Package cmd contains milestone commands
package cmd

import (
	"fmt"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/internal/utils"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var milestoneCmd = &cobra.Command{
	Use:     "milestone",
	Aliases: []string{"milestones"},
	Short:   "Manage milestones",
	Long:    `List, create, get, update, close, reopen, and delete milestones`,
}

var milestoneListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List milestones",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runMilestoneList,
}

var milestoneGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <milestone-id>",
	Short: "Get a specific milestone",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runMilestoneGet,
}

var milestoneCreateCmd = &cobra.Command{
	Use:   "create [<owner>/<repo>]",
	Short: "Create a new milestone",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runMilestoneCreate,
}

var milestoneUpdateCmd = &cobra.Command{
	Use:   "update [<owner>/<repo>] <milestone-id>",
	Short: "Update a milestone",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runMilestoneUpdate,
}

var milestoneCloseCmd = &cobra.Command{
	Use:   "close [<owner>/<repo>] <milestone-id>",
	Short: "Close a milestone",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runMilestoneClose,
}

var milestoneReopenCmd = &cobra.Command{
	Use:   "reopen [<owner>/<repo>] <milestone-id>",
	Short: "Reopen a closed milestone",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runMilestoneReopen,
}

var milestoneDeleteCmd = &cobra.Command{
	Use:   "delete [<owner>/<repo>] <milestone-id>",
	Short: "Delete a milestone",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runMilestoneDelete,
}

func init() {
	RootCmd.AddCommand(milestoneCmd)
	milestoneCmd.AddCommand(milestoneListCmd, milestoneGetCmd, milestoneCreateCmd, milestoneUpdateCmd, milestoneCloseCmd, milestoneReopenCmd, milestoneDeleteCmd)

	milestoneListCmd.Flags().StringP("state", "s", "", "Filter by state: open, closed, all")
	milestoneListCmd.Flags().Int("page", 0, "Page number")
	milestoneListCmd.Flags().Int("per-page", 0, "Results per page")

	milestoneCreateCmd.Flags().StringP("title", "t", "", "Milestone title (required)")
	milestoneCreateCmd.MarkFlagRequired("title")
	milestoneCreateCmd.Flags().StringP("description", "d", "", "Milestone description")
	milestoneCreateCmd.Flags().StringP("due-date", "D", "", "Due date (YYYY-MM-DD or YYYY-MM-DDTHH:MM:SSZ)")

	milestoneUpdateCmd.Flags().StringP("title", "t", "", "New milestone title")
	milestoneUpdateCmd.Flags().StringP("description", "d", "", "New milestone description")
	milestoneUpdateCmd.Flags().StringP("due-date", "D", "", "New due date (YYYY-MM-DD or YYYY-MM-DDTHH:MM:SSZ)")
}

func runMilestoneList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	listOpts := gitea.ListMilestoneOption{}

	state, _ := cmd.Flags().GetString("state")
	if state != "" {
		switch state {
		case "open", "closed", "all":
			listOpts.State = gitea.StateType(state)
		default:
			return errors.NewValidationError(fmt.Sprintf("invalid state: %s", state),
				map[string]interface{}{"valid": []string{"open", "closed", "all"}})
		}
	}

	milestones, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Milestone, *gitea.Response, error) {
		opts := listOpts
		opts.ListOptions = lo
		return client.ListRepoMilestones(owner, repo, opts)
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	return printer.Emit(milestones, func() error {
		if len(milestones) == 0 {
			printer.Println("No milestones found")
			return nil
		}

		headers := []string{"ID", "Title", "State", "Open", "Closed", "Due Date"}
		var rows [][]string

		for _, m := range milestones {
			dueDate := "-"
			if m.Deadline != nil {
				dueDate = m.Deadline.Format("2006-01-02")
			}

			rows = append(rows, []string{
				fmt.Sprintf("%d", m.ID),
				utils.Truncate(m.Title, 30),
				string(m.State),
				fmt.Sprintf("%d", m.OpenIssues),
				fmt.Sprintf("%d", m.ClosedIssues),
				dueDate,
			})
		}

		return printer.PrintTable(headers, rows)
	})
}

func runMilestoneGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	milestoneID := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	milestone, resp, err := client.GetMilestone(owner, repo, milestoneID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "milestone", fmt.Sprintf("%d", milestoneID))
	}

	printer := getPrinter(cmd)
	return printer.Emit(milestone, func() error {
		return printMilestoneText(printer, milestone)
	})
}

func runMilestoneCreate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}

	title, _ := cmd.Flags().GetString("title")
	description, _ := cmd.Flags().GetString("description")
	dueDateStr, _ := cmd.Flags().GetString("due-date")

	if title == "" {
		return errors.NewValidationError("--title is required",
			map[string]interface{}{"missing_flags": []string{"--title"}})
	}

	var dueDate *time.Time
	if dueDateStr != "" {
		parsed, err := parseDueDate(dueDateStr)
		if err != nil {
			return errors.NewValidationError(fmt.Sprintf("invalid due date format: %s", dueDateStr),
				map[string]interface{}{"expected": "YYYY-MM-DD or YYYY-MM-DDTHH:MM:SSZ"})
		}
		dueDate = &parsed
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.CreateMilestoneOption{
		Title:       title,
		Description: description,
		Deadline:    dueDate,
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would create milestone in %s\n", args[0])
		printer.Printf("  Title: %s\n", title)
		if description != "" {
			printer.Printf("  Description: %s\n", utils.Truncate(description, 80))
		}
		if dueDate != nil {
			printer.Printf("  Due Date: %s\n", dueDate.Format("2006-01-02"))
		}
		return nil
	}

	milestone, resp, err := client.CreateMilestone(owner, repo, opts)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(milestone, func() error {
		return printMilestoneText(printer, milestone)
	})
}

func runMilestoneUpdate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	milestoneID := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	if !cmd.Flags().Changed("title") && !cmd.Flags().Changed("description") && !cmd.Flags().Changed("due-date") {
		return errors.NewValidationError("at least one of --title, --description, or --due-date is required",
			map[string]interface{}{"required_flags": []string{"--title", "--description", "--due-date"}})
	}

	opts := gitea.EditMilestoneOption{}

	if cmd.Flags().Changed("title") {
		title, _ := cmd.Flags().GetString("title")
		opts.Title = title
	}

	if cmd.Flags().Changed("description") {
		desc, _ := cmd.Flags().GetString("description")
		opts.Description = &desc
	}

	if cmd.Flags().Changed("due-date") {
		dueDateStr, _ := cmd.Flags().GetString("due-date")
		parsed, err := parseDueDate(dueDateStr)
		if err != nil {
			return errors.NewValidationError(fmt.Sprintf("invalid due date format: %s", dueDateStr),
				map[string]interface{}{"expected": "YYYY-MM-DD or YYYY-MM-DDTHH:MM:SSZ"})
		}
		opts.Deadline = &parsed
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would update milestone #%d in %s\n", milestoneID, args[0])
		if cmd.Flags().Changed("title") {
			title, _ := cmd.Flags().GetString("title")
			printer.Printf("  Title: %s\n", title)
		}
		if cmd.Flags().Changed("description") {
			desc, _ := cmd.Flags().GetString("description")
			printer.Printf("  Description: %s\n", utils.Truncate(desc, 80))
		}
		if cmd.Flags().Changed("due-date") {
			dueDateStr, _ := cmd.Flags().GetString("due-date")
			parsed, _ := parseDueDate(dueDateStr)
			printer.Printf("  Due Date: %s\n", parsed.Format("2006-01-02"))
		}
		return nil
	}

	milestone, resp, err := client.EditMilestone(owner, repo, milestoneID, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "milestone", fmt.Sprintf("%d", milestoneID))
	}

	printer := getPrinter(cmd)
	return printer.Emit(milestone, func() error {
		return printMilestoneText(printer, milestone)
	})
}

func runMilestoneClose(cmd *cobra.Command, args []string) error {
	return updateMilestoneState(cmd, args, "closed")
}

func runMilestoneReopen(cmd *cobra.Command, args []string) error {
	return updateMilestoneState(cmd, args, "open")
}

func updateMilestoneState(cmd *cobra.Command, args []string, state string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	milestoneID := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	st := gitea.StateType(state)
	opts := gitea.EditMilestoneOption{
		State: &st,
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would %s milestone #%d in %s\n", state, milestoneID, args[0])
		return nil
	}

	milestone, resp, err := client.EditMilestone(owner, repo, milestoneID, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "milestone", fmt.Sprintf("%d", milestoneID))
	}

	printer := getPrinter(cmd)
	return printer.Emit(milestone, func() error {
		printer.Printf("%s milestone #%d\n", strings.ToUpper(state), milestone.ID)
		return nil
	})
}

func runMilestoneDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	milestoneID := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would delete milestone #%d in %s\n", milestoneID, args[0])
		return nil
	}

	resp, err := client.DeleteMilestone(owner, repo, milestoneID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "milestone", fmt.Sprintf("%d", milestoneID))
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"status": "deleted", "milestone": milestoneID}, func() error {
		printer.Printf("Deleted milestone #%d\n", milestoneID)
		return nil
	})
}

func parseDueDate(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}

	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}
