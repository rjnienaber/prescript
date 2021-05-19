package lib

import (
	"github.com/spf13/cobra"
	"time"
)

type subcommand int

const (
	NoCommand subcommand = iota
	PlayCommand
	RecordCommand
)

type PlayConfig struct {
	Quiet          bool
	DontFail       bool
	LogLevel       string
	Timeout        time.Duration
	ScriptFile     string
	ExecutablePath string
}

type RecordConfig struct {
	IgnoreOutput   bool
	ScriptFile     string
	ExecutablePath string
	Arguments      []string
}

type Config struct {
	Subcommand subcommand
	Play       PlayConfig
	Record     RecordConfig
	Logger     Logger
}

func createPlaySubCommand(config *Config) *cobra.Command {
	var playCmd = &cobra.Command{
		Use:   "play [script file] [optional executable]",
		Short: "runs prescripted responses against an interactive cli",
		Long:  "runs through a predefined script of responses to an interactive cli",
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

	defaultTimeout, _ := time.ParseDuration("30s")
	playCmd.Flags().DurationVarP(&config.Play.Timeout, "timeout", "t", defaultTimeout, "timeout waiting for output from external command")
	return playCmd
}

func createRecordSubCommand(config *Config) *cobra.Command {
	var recordCmd = &cobra.Command{
		Use:   "record [script file] [optional executable]",
		Short: "runs a cli and records ouput and responses",
		Long:  "runs a cli and records ouput and responses",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			config.Subcommand = RecordCommand
			config.Record.ScriptFile = args[0]
			config.Record.ExecutablePath = args[1]
			config.Record.Arguments = args[2:]
		},
	}

	recordCmd.Flags().BoolVarP(&config.Record.IgnoreOutput, "ignoreOutput", "i", false, "ignore output from external command")
	return recordCmd
}

func GetConfig() (Config, error) {
	config := Config{}

	playCmd := createPlaySubCommand(&config)
	recordCmd := createRecordSubCommand(&config)

	var rootCmd = &cobra.Command{
		Use:   "prescript [cmd] [script file] [optional executable]",
		Short: "record and playback responses to an interactive cli",
		Long:  "record and playback responses to an interactive cli",
	}
	rootCmd.Args = cobra.MinimumNArgs(1)
	rootCmd.AddCommand(playCmd)
	rootCmd.AddCommand(recordCmd)

	err := rootCmd.Execute()
	if err != nil {
		return Config{}, err
	}

	return config, nil
}
