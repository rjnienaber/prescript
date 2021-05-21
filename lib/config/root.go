package config

import (
	"github.com/spf13/cobra"
)

type Logger interface {
	Close()
	Debug(args ...interface{})
	Debugf(template string, args ...interface{})
	Error(args ...interface{})
	Info(args ...interface{})
}

type subcommand int

const (
	NoCommand subcommand = iota
	PlayCommand
	RecordCommand
)

type Config struct {
	Subcommand subcommand
	Play       PlayConfig
	Record     RecordConfig
	Logger     Logger
}

func GetConfig() (Config, error) {
	config := Config{}

	playCmd := createPlaySubCommand(&config)
	recordCmd := createRecordSubCommand(&config)

	var rootCmd = &cobra.Command{
		Use:   "prescript [cmd] [script file] [optional executable]",
		Short: "Record and playback responses to an interactive cli",
		Long:  "Record and playback responses to an interactive cli",
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
