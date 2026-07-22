package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/meigma/packages/internal/githubrelease"
)

func newFetchReleaseCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "fetch-release",
		Short: "Download and verify a registered GitHub Release",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			request := githubrelease.Request{
				RegistryPath: options.Viper.GetString("registry"),
				Project:      options.Viper.GetString("project"),
				Tag:          options.Viper.GetString("tag"),
				OutputDir:    options.Viper.GetString("output"),
				Token:        options.Viper.GetString("github-token"),
			}
			result, err := options.FetchRelease(command.Context(), request)
			if err != nil {
				return fmt.Errorf("fetch release: %w", err)
			}
			if err := json.NewEncoder(options.Out).Encode(result); err != nil {
				return fmt.Errorf("write fetch result: %w", err)
			}

			return nil
		},
	}

	flags := command.Flags()
	flags.String("registry", "", "path to the canonical project registry")
	flags.String("project", "", "registered project to fetch")
	flags.String("tag", "", "stable v-prefixed source release tag")
	flags.String("output", "", "new directory for verified release assets")

	return command
}
