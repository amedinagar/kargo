package logout

import (
	"fmt"
	"log"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/spf13/cobra"
)

func NewCommand(opt *option.Option) *cobra.Command {

	cmd := &cobra.Command{
		Use:     "logout CONTEXT",
		Short:   "Logout from Kargo API server",
		Example: "kargo logout",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()

			if err := config.DeleteCLIConfig(); err != nil {
				log.Fatalf("Error deleting configuration: %v", err)
			}

			fmt.Printf("Logged out from '%s'\n", ctx)
		},
	}
	return cmd

}
