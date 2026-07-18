package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/meigma/packages/internal/localrepo"
)

func newRebuildLocalCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "rebuild-local",
		Short: "Rebuild a deterministic candidate from fixture release sets",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			request := localrepo.RebuildRequest{
				RegistryPath:          options.Viper.GetString("registry"),
				Project:               options.Viper.GetString("project"),
				ReleasesDir:           options.Viper.GetString("releases"),
				Root:                  options.Viper.GetString("root"),
				GNUPGHome:             options.Viper.GetString("gnupg-home"),
				SigningKey:            options.Viper.GetString("signing-key"),
				SigningPassphrase:     options.Viper.GetString("gpg-passphrase"),
				SigningPassphraseFile: options.Viper.GetString("gpg-passphrase-file"),
				BaseURL:               options.Viper.GetString("base-url"),
			}
			result, err := options.RebuildLocal(command.Context(), request)
			if err != nil {
				return fmt.Errorf("rebuild local candidate: %w", err)
			}
			if err := json.NewEncoder(options.Out).Encode(result); err != nil {
				return fmt.Errorf("write rebuild result: %w", err)
			}

			return nil
		},
	}

	flags := command.Flags()
	flags.String("registry", "", "path to the fixture project registry")
	flags.String("project", "", "registry project to rebuild")
	flags.String("releases", "", "directory containing v-prefixed fixture release directories")
	flags.String("root", "", "candidate tree to create or verify as a no-op")
	flags.String("gnupg-home", "", "signing keyring directory")
	flags.String("signing-key", "", "full signing-subkey fingerprint")
	flags.String("gpg-passphrase-file", "", "mode-0600 file containing the signing passphrase")
	flags.String("base-url", "", "public root URL rendered into install configuration")

	return command
}
