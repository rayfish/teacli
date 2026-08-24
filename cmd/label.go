// Package cmd contains label commands
package cmd

import (
	"strconv"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/internal/utils"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var labelCmd = &cobra.Command{
	Use:     "label",
	Aliases: []string{"labels"},
	Short:   "Manage labels",
	Long:    `Manage labels in repositories`,
}

var labelListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List labels",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLabelList,
}

var labelGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <label-id>",
	Short: "Get a specific label",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runLabelGet,
}

var labelCreateCmd = &cobra.Command{
	Use:   "create [<owner>/<repo>]",
	Short: "Create a new label",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLabelCreate,
}

var labelUpdateCmd = &cobra.Command{
	Use:   "update [<owner>/<repo>] <label-id>",
	Short: "Update a label",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runLabelUpdate,
}

var labelDeleteCmd = &cobra.Command{
	Use:   "delete [<owner>/<repo>] <label-id>",
	Short: "Delete a label",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runLabelDelete,
}

func init() {
	RootCmd.AddCommand(labelCmd)
	labelCmd.AddCommand(labelListCmd, labelGetCmd, labelCreateCmd, labelUpdateCmd, labelDeleteCmd)

	labelCreateCmd.Flags().StringP("name", "n", "", "Label name (required)")
	labelCreateCmd.MarkFlagRequired("name")
	labelCreateCmd.Flags().StringP("color", "c", "", "Label color (hex code without #) (required)")
	labelCreateCmd.MarkFlagRequired("color")
	labelCreateCmd.Flags().StringP("description", "d", "", "Label description")
	labelCreateCmd.Flags().Bool("exclusive", false, "Make label exclusive")

	labelUpdateCmd.Flags().StringP("name", "n", "", "New label name")
	labelUpdateCmd.Flags().StringP("color", "c", "", "New label color (hex code without #)")
	labelUpdateCmd.Flags().StringP("description", "d", "", "New label description")
	labelUpdateCmd.Flags().Bool("exclusive", false, "Set label as exclusive")
	labelUpdateCmd.Flags().Bool("no-exclusive", false, "Remove exclusive flag from label")

	labelDeleteCmd.Flags().BoolP("yes", "y", false, "No-op, kept for compatibility: delete never prompts")

	labelListCmd.Flags().Int("page", 0, "Page number")
	labelListCmd.Flags().Int("per-page", 0, "Results per page")
}

func runLabelList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	labels, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Label, *gitea.Response, error) {
		return client.ListRepoLabels(owner, repo, gitea.ListLabelsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	headers := []string{"ID", "Name", "Color", "Description", "Exclusive", "Archived"}
	var rows [][]string
	for _, label := range labels {
		rows = append(rows, []string{
			strconv.FormatInt(label.ID, 10),
			label.Name,
			label.Color,
			label.Description,
			strconv.FormatBool(label.Exclusive),
			strconv.FormatBool(label.IsArchived),
		})
	}

	return printer.Emit(labels, func() error {
		if len(labels) == 0 {
			printer.Println("No labels found")
			return nil
		}
		return printer.PrintTable(headers, rows)
	})
}

func runLabelGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	labelID := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	label, resp, err := client.GetRepoLabel(owner, repo, labelID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "label", strconv.FormatInt(labelID, 10))
	}

	return printer.Emit(label, func() error {
		printer.Printf("Label Details:\n")
		printer.Printf("--------------\n")
		printer.Printf("ID:        %d\n", label.ID)
		printer.Printf("Name:      %s\n", label.Name)
		printer.Printf("Color:     #%s\n", label.Color)
		printer.Printf("Exclusive: %t\n", label.Exclusive)
		printer.Printf("Archived:  %t\n", label.IsArchived)
		if label.Description != "" {
			printer.Printf("Description: %s\n", label.Description)
		}
		return nil
	})
}

func runLabelCreate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	color, _ := cmd.Flags().GetString("color")
	description, _ := cmd.Flags().GetString("description")
	exclusive, _ := cmd.Flags().GetBool("exclusive")

	if name == "" {
		return errors.NewValidationError("label name is required",
			map[string]interface{}{"hint": "Use --name or -n flag"})
	}

	if color == "" {
		return errors.NewValidationError("label color is required",
			map[string]interface{}{"hint": "Use --color or -c flag (hex code without #)"})
	}

	if !isValidColor(color) {
		return errors.NewValidationError("invalid color format",
			map[string]interface{}{"provided": color, "expected": "hex code (e.g., 00aabb)"})
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer.Printf("DRY-RUN: Would create label '%s' (#%s) in %s/%s\n", name, color, owner, repo)
		return nil
	}

	opt := gitea.CreateLabelOption{
		Name:        name,
		Color:       color,
		Description: description,
		Exclusive:   exclusive,
	}

	label, resp, err := client.CreateLabel(owner, repo, opt)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	headers := []string{"ID", "Name", "Color", "Description", "Exclusive", "Archived"}
	rows := [][]string{{
		strconv.FormatInt(label.ID, 10),
		label.Name,
		label.Color,
		label.Description,
		strconv.FormatBool(label.Exclusive),
		strconv.FormatBool(label.IsArchived),
	}}

	return printer.Emit(label, func() error {
		printer.Printf("Label created: %s (#%s) [ID: %d]\n", label.Name, label.Color, label.ID)
		return printer.PrintTable(headers, rows)
	})
}

func runLabelUpdate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	labelID := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	opt := gitea.EditLabelOption{}
	changed := false

	if cmd.Flags().Changed("name") {
		name, _ := cmd.Flags().GetString("name")
		opt.Name = &name
		changed = true
	}

	if cmd.Flags().Changed("color") {
		color, _ := cmd.Flags().GetString("color")
		if !isValidColor(color) {
			return errors.NewValidationError("invalid color format",
				map[string]interface{}{"provided": color, "expected": "hex code (e.g., 00aabb)"})
		}
		opt.Color = &color
		changed = true
	}

	if cmd.Flags().Changed("description") {
		desc, _ := cmd.Flags().GetString("description")
		opt.Description = &desc
		changed = true
	}

	if cmd.Flags().Changed("exclusive") {
		exclusive, _ := cmd.Flags().GetBool("exclusive")
		opt.Exclusive = &exclusive
		changed = true
	}

	if cmd.Flags().Changed("no-exclusive") {
		noExclusive, _ := cmd.Flags().GetBool("no-exclusive")
		exclusive := !noExclusive
		opt.Exclusive = &exclusive
		changed = true
	}

	if !changed {
		return errors.NewValidationError("no update options provided",
			map[string]interface{}{
				"available_options": []string{"--name", "--color", "--description", "--exclusive", "--no-exclusive"},
			})
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer.Printf("DRY-RUN: Would update label %d in %s/%s\n", labelID, owner, repo)
		if opt.Name != nil {
			printer.Printf("  Name: %s\n", *opt.Name)
		}
		if opt.Color != nil {
			printer.Printf("  Color: %s\n", *opt.Color)
		}
		if opt.Description != nil {
			printer.Printf("  Description: %s\n", *opt.Description)
		}
		if opt.Exclusive != nil {
			printer.Printf("  Exclusive: %t\n", *opt.Exclusive)
		}
		return nil
	}

	label, resp, err := client.EditLabel(owner, repo, labelID, opt)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "label", strconv.FormatInt(labelID, 10))
	}

	headers := []string{"ID", "Name", "Color", "Description", "Exclusive", "Archived"}
	rows := [][]string{{
		strconv.FormatInt(label.ID, 10),
		label.Name,
		label.Color,
		label.Description,
		strconv.FormatBool(label.Exclusive),
		strconv.FormatBool(label.IsArchived),
	}}

	return printer.Emit(label, func() error {
		printer.Printf("Label updated: %s (#%s) [ID: %d]\n", label.Name, label.Color, label.ID)
		return printer.PrintTable(headers, rows)
	})
}

func runLabelDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	labelID := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would delete label %d in %s/%s\n", labelID, owner, repo)
		return nil
	}

	resp, err := client.DeleteLabel(owner, repo, labelID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "label", strconv.FormatInt(labelID, 10))
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"status": "deleted", "label": labelID}, func() error {
		printer.Printf("Label %d deleted successfully\n", labelID)
		return nil
	})
}

func isValidColor(color string) bool {
	if len(color) != 6 {
		return false
	}
	for _, c := range color {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
