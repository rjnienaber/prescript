package lib

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"time"
)

type subcommand int

const (
	NotSpecified subcommand = iota
	Play
	Record
)

type PlayConfig struct {
	Quiet      bool
	DontFail   bool
	Verbose    bool
	Timeout    time.Duration
	ScriptFile string
	Executable string
}

type RecordConfig struct {
	IgnoreOutput bool
	ScriptFile   string
	Executable   string
	Arguments    []string
}

type Config struct {
	Subcommand subcommand
	Play       PlayConfig
	Record     RecordConfig
	Logger     *zap.SugaredLogger
}

func createPlaySubCommand(config *Config) *cobra.Command {
	var playCmd = &cobra.Command{
		Use:   "play [script file] [optional executable]",
		Short: "runs prescripted responses against an interactive cli",
		Long:  "runs through a predefined script of responses to an interactive cli",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			config.Subcommand = Play
			config.Play.ScriptFile = args[0]
			if len(args) > 1 {
				config.Play.Executable = args[1]
			}
		},
	}

	playCmd.Flags().BoolVarP(&config.Play.Quiet, "quiet", "q", false, "quiet mode, no output")
	playCmd.Flags().BoolVarP(&config.Play.DontFail, "dont-fail", "d", false, "dont fail on external command failures")
	playCmd.Flags().BoolVarP(&config.Play.Verbose, "verbose", "v", false, "output all logs")

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
			config.Subcommand = Record
			config.Record.ScriptFile = args[0]
			config.Record.Executable = args[1]
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
