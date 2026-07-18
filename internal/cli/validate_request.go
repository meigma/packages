package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newValidateRequestCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "validate-request",
		Short: "Validate an unprivileged publish or rebuild request",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			result, err := options.ValidateRequest(
				options.Viper.GetString("registry"),
				options.Viper.GetString("project"),
				options.Viper.GetString("tag"),
			)
			if err != nil {
				return fmt.Errorf("validate workflow request: %w", err)
			}
			if err := json.NewEncoder(options.Out).Encode(result); err != nil {
				return fmt.Errorf("write validation result: %w", err)
			}

			return nil
		},
	}

	flags := command.Flags()
	flags.String("registry", "", "path to the project registry")
	flags.String("project", "", "registry project to validate")
	flags.String("tag", "", "optional stable v-prefixed release tag")

	return command
}
