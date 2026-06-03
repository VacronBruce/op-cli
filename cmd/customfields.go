package cmd

import (
	"fmt"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

// customFieldLinks resolves option names for a logical custom field into links,
// returning the field key (e.g. "customField12") and the resolved links. Used
// by create/update for multi-value custom fields.
func customFieldLinks(fieldName string, names []string) (string, []api.Link, error) {
	cf, err := api.CustomFieldByName(fieldName)
	if err != nil {
		return "", nil, err
	}
	links := make([]api.Link, 0, len(names))
	for _, n := range names {
		href, err := cf.ResolveHref(n)
		if err != nil {
			return "", nil, fmt.Errorf("resolving %s: %w", fieldName, err)
		}
		links = append(links, api.Link{Href: href})
	}
	return cf.Field, links, nil
}

// customFieldFilter resolves an option name to (fieldKey, optionID) for use as a
// work-package list filter. Used by board/my/check.
func customFieldFilter(fieldName, value string) (string, string, error) {
	cf, err := api.CustomFieldByName(fieldName)
	if err != nil {
		return "", "", err
	}
	id, err := cf.OptionID(value)
	if err != nil {
		return "", "", fmt.Errorf("resolving %s: %w", fieldName, err)
	}
	return cf.Field, id, nil
}

// completeCustomField returns a cobra completion function that suggests the
// option names of a logical custom field (honoring ~/.oprc overrides).
func completeCustomField(fieldName string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		cf, err := api.CustomFieldByName(fieldName)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return cf.OptionNames(), cobra.ShellCompDirectiveNoFileComp
	}
}
