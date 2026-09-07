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
	RunnerFile     string
	Terminal       string
	Tape           string
	Arguments      []string
}

func createPlaySubCommand(config *Config) *cobra.Command {
	var playCmd = &cobra.Command{
		Use:   "play [script file] [--runner runner file] [-- executable arguments]",
		Short: "Runs prescripted responses against an interactive cli",
		Long:  "Runs through a predefined script of responses to an interactive cli",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			config.Subcommand = PlayCommand
			config.Play.ScriptFile = args[0]
			config.Play.Arguments = args[1:]
		},
	}

	playCmd.Flags().BoolVarP(&config.Play.Quiet, "quiet", "q", false, "quiet mode, no output")
	playCmd.Flags().BoolVarP(&config.Play.DontFail, "dont-fail", "d", false, "dont fail on external command failures")
	playCmd.Flags().StringVarP(&config.Play.LogLevel, "log-level", "l", "none", "log level to use with logs (none, error, info or debug)")
	playCmd.Flags().StringVarP(&config.Play.ExecutablePath, "exec", "e", "", "override the executable named in the script file")
	playCmd.Flags().StringVarP(&config.Play.RunnerFile, "runner", "r", "", "runner file describing how to launch an implementation")
	playCmd.Flags().StringVar(&config.Play.Terminal, "terminal", "", "override what the executable is given for its standard streams (pty or pipes)")
	playCmd.Flags().StringVar(&config.Play.Tape, "tape", "", "file of recorded random values to replay to the executable, instead of letting it draw its own")

	defaultTimeout := 30 * time.Second
	playCmd.Flags().DurationVarP(&config.Play.Timeout, "timeout", "t", defaultTimeout, "timeout waiting for output from external command")
	return playCmd
}
