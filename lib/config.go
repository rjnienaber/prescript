package lib

import (
	"github.com/spf13/cobra"
	"time"
)

type subcommand int

const (
	Play subcommand = iota
	Record
)

type PlayConfig struct {
	Quiet      bool
	DontFail   bool
	Verbose    bool
	Timeout    time.Duration
	ScriptFile string
	Command    string
}

type Config struct {
	Subcommand subcommand
	Play       PlayConfig
}

func createPlaySubCommand(config *Config, defaultTimeout time.Duration) *cobra.Command {
	var playCmd = &cobra.Command{
		Use:   "play [script file] [optional command]",
		Short: "runs prescripted commands against an interactive cli",
		Long:  "runs through a predefined script of responses to an interactive cli",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			config.Play.ScriptFile = args[0]
			if len(args) > 1 {
				config.Play.Command = args[1]
			}
		},
	}

	playCmd.Flags().BoolVarP(&config.Play.Quiet, "quiet", "q", false, "quiet mode, no output")
	playCmd.Flags().BoolVarP(&config.Play.DontFail, "dont-fail", "d", false, "dont fail on external command failures")
	playCmd.Flags().BoolVarP(&config.Play.Verbose, "verbose", "v", false, "output all logs")
	playCmd.Flags().DurationVarP(&config.Play.Timeout, "timeout", "t", defaultTimeout, "timeout waiting for output from external command")
	return playCmd
}

func GetConfig() (*Config, error) {
	config := Config{}

	defaultTimeout, _ := time.ParseDuration("30s")

	playCmd := createPlaySubCommand(&config, defaultTimeout)

	var rootCmd = &cobra.Command{
		Use:   "prescript [cmd] [script file] [command]",
		Short: "record and playback responses to an interactive cli",
		Long:  "record and playback responses to an interactive cli",
	}
	rootCmd.AddCommand(playCmd)

	err := rootCmd.Execute()
	if err != nil {
		return &Config{}, err
	}

	return &config, nil
}
