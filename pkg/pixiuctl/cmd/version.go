package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	// Version is the pixiuctl client version.
	Version = "pixiuctl version 0.2.4"
)

// NewVersionCommand prints the pixiuctl version.
func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the pixiuctl version",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), Version)
			return err
		},
	}

	return cmd
}
