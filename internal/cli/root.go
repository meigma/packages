package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/meigma/packages/internal/localrepo"
)

// LocalBuildFunc builds one verified local candidate tree.
type LocalBuildFunc func(context.Context, localrepo.Request) (localrepo.Result, error)

// BuildInfo describes build metadata printed by --version.
type BuildInfo struct {
	// Version identifies the CLI build.
	Version string
	// Commit is the source commit used to build the binary.
	Commit string
	// Date is the build timestamp.
	Date string
}

// Options customizes root command construction.
type Options struct {
	// In receives interactive command input.
	In io.Reader
	// Out receives machine-readable command output.
	Out io.Writer
	// Err receives diagnostics and human-readable status.
	Err io.Writer
	// Build controls the root command version output.
	Build BuildInfo
	// Viper is the configuration instance used by the command tree.
	Viper *viper.Viper
	// BuildLocal supplies the fixture-to-candidate implementation.
	BuildLocal LocalBuildFunc
}

// NewRootCommand creates the meigma-packages Cobra command tree.
func NewRootCommand(options Options) *cobra.Command {
	if options.In == nil {
		options.In = strings.NewReader("")
	}
	if options.Out == nil {
		options.Out = io.Discard
	}
	if options.Err == nil {
		options.Err = io.Discard
	}
	if options.Viper == nil {
		options.Viper = viper.New()
	}
	if options.BuildLocal == nil {
		options.BuildLocal = localrepo.Build
	}
	options.Build = options.Build.withDefaults()

	root := &cobra.Command{
		Use:           "meigma-packages",
		Short:         "Build and publish Meigma APT and RPM repositories",
		Version:       options.Build.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return initializeConfig(cmd, options.Viper)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	root.SetVersionTemplate(
		fmt.Sprintf(
			"meigma-packages %s (%s) built %s\n",
			options.Build.Version,
			options.Build.Commit,
			options.Build.Date,
		),
	)
	root.SetIn(options.In)
	root.SetOut(options.Out)
	root.SetErr(options.Err)
	root.AddCommand(newBuildLocalCommand(options))

	return root
}

func (b BuildInfo) withDefaults() BuildInfo {
	if strings.TrimSpace(b.Version) == "" {
		b.Version = "dev"
	}
	if strings.TrimSpace(b.Commit) == "" {
		b.Commit = "none"
	}
	if strings.TrimSpace(b.Date) == "" {
		b.Date = "unknown"
	}

	return b
}

func initializeConfig(cmd *cobra.Command, vp *viper.Viper) error {
	vp.SetEnvPrefix("MEIGMA_PACKAGES")
	vp.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	vp.AutomaticEnv()

	if err := vp.BindPFlags(cmd.Root().PersistentFlags()); err != nil {
		return fmt.Errorf("bind persistent flags: %w", err)
	}
	if err := vp.BindPFlags(cmd.Flags()); err != nil {
		return fmt.Errorf("bind flags: %w", err)
	}

	return nil
}
