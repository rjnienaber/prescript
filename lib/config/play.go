package config

import (
	"time"

	"github.com/spf13/cobra"
)

type PlayConfig struct {
	Quiet          bool
	DontFail       bool
	LogLevel       string
	Timeout        time.Duration
	ScriptFile     string
	ExecutablePath string
}

func createPlaySubCommand(config *Config) *cobra.Command {
	var playCmd = &cobra.Command{
		Use:   "play [script file] [optional executable]",
		Short: "Runs prescripted responses against an interactive cli",
		Long:  "Runs through a predefined script of responses to an interactive cli",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			config.Subcommand = PlayCommand
			config.Play.ScriptFile = args[0]
			if len(args) > 1 {
				config.Play.ExecutablePath = args[1]
			}
		},
	}

	playCmd.Flags().BoolVarP(&config.Play.Quiet, "quiet", "q", false, "quiet mode, no output")
	playCmd.Flags().BoolVarP(&config.Play.DontFail, "dont-fail", "d", false, "dont fail on external command failures")
	playCmd.Flags().StringVarP(&config.Play.LogLevel, "log-level", "l", "none", "log level to use with logs (e.g. none, debug, info)")

	defaultTimeout := 30 * time.Second
	playCmd.Flags().DurationVarP(&config.Play.Timeout, "timeout", "t", defaultTimeout, "timeout waiting for output from external command")
	return playCmd
}
