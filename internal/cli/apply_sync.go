package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/meigma/packages/internal/r2repo"
)

func newApplySyncCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "apply-sync",
		Short: "Apply and verify an ordered candidate-tree sync to R2",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			result, err := options.ApplySync(command.Context(), r2repo.Request{
				Root:            options.Viper.GetString("root"),
				Bucket:          options.Viper.GetString("bucket"),
				Prefix:          options.Viper.GetString("prefix"),
				Endpoint:        options.Viper.GetString("endpoint"),
				AccessKeyID:     options.Viper.GetString("r2-access-key-id"),
				SecretAccessKey: options.Viper.GetString("r2-secret-access-key"),
				SessionToken:    options.Viper.GetString("r2-session-token"),
			})
			if err != nil {
				return fmt.Errorf("apply sync: %w", err)
			}
			if err := json.NewEncoder(options.Out).Encode(result); err != nil {
				return fmt.Errorf("write sync result: %w", err)
			}

			return nil
		},
	}

	flags := command.Flags()
	flags.String("root", "", "verified candidate tree")
	flags.String("bucket", "", "R2 bucket")
	flags.String("prefix", "", "confined R2 object prefix")
	flags.String("endpoint", "", "R2 S3 endpoint")

	return command
}
