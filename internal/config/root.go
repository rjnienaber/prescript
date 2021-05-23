package config

import (
	"github.com/rjnienaber/prescript/internal/utils"
	"github.com/spf13/cobra"
)

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
	Logger     utils.Logger
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

	rootCmd.AddCommand(playCmd)
	rootCmd.AddCommand(recordCmd)

	err := rootCmd.Execute()
	if err != nil {
		return Config{}, err
	}

	return config, nil
}
