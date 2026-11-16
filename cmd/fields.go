package cmd

import (
	"fmt"
	"strings"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

var fieldsCmd = &cobra.Command{
	Use:        "fields [name]",
	SuggestFor: []string{"statuses", "status"},
	Short:      "List custom fields and their allowed values",
	Long: `List the logical custom fields behind --component, --product,
--tech-area, --label, and --jira-id, or the allowed values of one field.

Values come from the built-in registry, overridden by the custom_fields:
section of ~/.oprc — so this shows exactly what create/update/board accept.

Examples:
  op fields              List all custom fields
  op fields component    List the allowed --component values`,
	Args:        cobra.MaximumNArgs(1),
	Annotations: skipClientInit(),
	RunE:        runFields,
}

func init() {
	rootCmd.AddCommand(fieldsCmd)
}

func runFields(cmd *cobra.Command, args []string) error {
	if len(args) == 1 {
		return describeField(args[0])
	}

	names := api.CustomFieldNames()
	fmt.Printf("Custom fields (%d):\n", len(names))
	for _, name := range names {
		cf, err := api.CustomFieldByName(name)
		if err != nil {
			return err
		}
		if n := len(cf.OptionNames()); n > 0 {
			fmt.Printf("  %-12s %-15s %d options\n", name, cf.Field, n)
		} else {
			fmt.Printf("  %-12s %-15s free text\n", name, cf.Field)
		}
	}
	fmt.Println("\nUse 'op fields <name>' to list a field's allowed values.")
	fmt.Println("Statuses are built in, not custom fields: op fields status explains them.")
	return nil
}

func describeField(name string) error {
	if strings.EqualFold(name, "status") {
		fmt.Println("status is a built-in OpenProject field, not a custom field.")
		fmt.Println("Set it with: op update <id> --status=\"<name>\"")
		fmt.Println("List the allowed values with: op api statuses")
		return nil
	}
	cf, err := api.CustomFieldByName(name)
	if err != nil {
		return err
	}
	options := cf.OptionNames()
	if len(options) == 0 {
		fmt.Printf("%s (%s): free-text field, no fixed values\n", name, cf.Field)
		return nil
	}
	fmt.Printf("%s (%s), %d allowed values:\n", name, cf.Field, len(options))
	for _, opt := range options {
		fmt.Printf("  %s\n", opt)
	}
	return nil
}
