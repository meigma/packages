package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/meigma/packages/internal/localrepo"
)

func newBuildLocalCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "build-local",
		Short: "Build and verify a candidate repository from fixture release assets",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			request := localrepo.Request{
				RegistryPath:          options.Viper.GetString("registry"),
				Project:               options.Viper.GetString("project"),
				ReleaseDir:            options.Viper.GetString("release"),
				Root:                  options.Viper.GetString("root"),
				GNUPGHome:             options.Viper.GetString("gnupg-home"),
				SigningKey:            options.Viper.GetString("signing-key"),
				SigningPassphrase:     options.Viper.GetString("gpg-passphrase"),
				SigningPassphraseFile: options.Viper.GetString("gpg-passphrase-file"),
				BaseURL:               options.Viper.GetString("base-url"),
			}
			result, err := options.BuildLocal(command.Context(), request)
			if err != nil {
				return fmt.Errorf("build local candidate: %w", err)
			}

			encoder := json.NewEncoder(options.Out)
			encoder.SetEscapeHTML(false)
			if err := encoder.Encode(result); err != nil {
				return fmt.Errorf("write build result: %w", err)
			}

			return nil
		},
	}

	flags := command.Flags()
	flags.String("registry", "", "path to the fixture project registry")
	flags.String("project", "", "registry project to build")
	flags.String("release", "", "directory containing fixture release assets")
	flags.String("root", "", "new candidate-tree output directory")
	flags.String("gnupg-home", "", "signing keyring directory")
	flags.String("signing-key", "", "full signing-subkey fingerprint")
	flags.String("gpg-passphrase-file", "", "mode-0600 file containing the signing passphrase")
	flags.String("base-url", "", "public root URL rendered into install configuration")

	return command
}
