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
	Quiet    bool
	DontFail bool
	Verbose  bool
	Timeout  time.Duration
}

type Config struct {
	Subcommand subcommand
	Play       PlayConfig
}

func createPlaySubCommand(config *Config, defaultTimeout time.Duration) *cobra.Command {
	var playCmd = &cobra.Command{
		Use:   "play [optional args] [script file] [executable]",
		Short: "runs prescripted commands against an interactive cli",
		Long:  "runs through a predefined script of responses to an interactive cli",
		Run: func(cmd *cobra.Command, args []string) {
			// left empty on purpose otherwise default help is displayed
		},
	}

	playCmd.PersistentFlags().BoolVarP(&config.Play.Quiet, "quiet", "q", false, "quiet mode, no output")
	playCmd.PersistentFlags().BoolVarP(&config.Play.DontFail, "dont-fail", "d", false, "dont fail on external command failures")
	playCmd.PersistentFlags().BoolVarP(&config.Play.Verbose, "verbose", "v", false, "output all logs")
	playCmd.PersistentFlags().DurationVarP(&config.Play.Timeout, "timeout", "t", defaultTimeout, "timeout waiting for output from external command")
	return playCmd
}

func GetConfig() *Config {
	config := Config{}

	defaultTimeout, _ := time.ParseDuration("30s")

	playCmd := createPlaySubCommand(&config, defaultTimeout)

	var rootCmd = &cobra.Command{
		Use:   "prescript [cmd] [optional args] [script file] [executable]",
		Short: "record and playback responses to an interactive cli",
		Long:  "record and playback responses to an interactive cli",
	}
	rootCmd.AddCommand(playCmd)

	cobra.CheckErr(rootCmd.Execute())
	return &config
}
