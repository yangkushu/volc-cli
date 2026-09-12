package cr

import "github.com/spf13/cobra"

func newDeleteTagsCmd() *cobra.Command {
	return &cobra.Command{Use: "delete-tags", Short: "DeleteTags(待实现)", RunE: func(*cobra.Command, []string) error { return nil }}
}
