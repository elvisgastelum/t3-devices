// Package cli defines the t3-devices command line.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/elvisgastelum/t3-devices/internal/logx"
	"github.com/elvisgastelum/t3-devices/internal/tui"
	"github.com/elvisgastelum/t3-devices/internal/version"
)

// runTUI and initLogging are swapped in tests to avoid starting the TUI or logging setup.
var (
	runTUI      = tui.Run
	initLogging = logx.Init
)

// versionOutput is the text printed by --version and the version command.
func versionOutput() string {
	return version.String() + "\n" + version.UpdateHint + "\n"
}

// NewRootCmd builds the root command with its flags and subcommands.
func NewRootCmd() *cobra.Command {
	var debug bool

	cmd := &cobra.Command{
		Use:          "t3-devices",
		Short:        "Manage T3 devices from the terminal",
		Version:      version.Version,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			initLogging(debug)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI()
		},
	}
	cmd.SetVersionTemplate(versionOutput())
	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")
	cmd.AddCommand(newVersionCmd())

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprint(cmd.OutOrStdout(), versionOutput())
			return err
		},
	}
}

// Execute runs the root command.
func Execute() error {
	return NewRootCmd().Execute()
}
