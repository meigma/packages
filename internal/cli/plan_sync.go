package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newPlanSyncCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "plan-sync",
		Short: "Plan ordered filesystem changes without applying them",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			plan, err := options.PlanSync(
				options.Viper.GetString("root"),
				options.Viper.GetString("remote"),
			)
			if err != nil {
				return fmt.Errorf("plan sync: %w", err)
			}
			if err := json.NewEncoder(options.Out).Encode(plan); err != nil {
				return fmt.Errorf("write sync plan: %w", err)
			}

			return nil
		},
	}

	flags := command.Flags()
	flags.String("root", "", "verified candidate tree")
	flags.String("remote", "", "existing filesystem tree to compare")

	return command
}
