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
	// Handled is set when cobra has already answered the command line itself
	// -- `--version` prints and there is nothing left to run. It is not a
	// failure, and a caller that treats "no subcommand" as a usage error would
	// otherwise exit non-zero having done exactly what was asked.
	Handled bool

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
		// The same string a generated bug report names itself by, so someone
		// reading one can check what they have against what made it.
		Version: utils.Version(),
	}

	rootCmd.AddCommand(playCmd)
	rootCmd.AddCommand(recordCmd)

	err := rootCmd.Execute()
	if err != nil {
		return Config{}, err
	}

	if asked, _ := rootCmd.Flags().GetBool("version"); asked {
		config.Handled = true
	}

	return config, nil
}
