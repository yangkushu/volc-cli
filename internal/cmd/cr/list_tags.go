package cr

import "github.com/spf13/cobra"

func newListTagsCmd() *cobra.Command {
	return &cobra.Command{Use: "list-tags", Short: "ListTags(待实现)", RunE: func(*cobra.Command, []string) error { return nil }}
}
