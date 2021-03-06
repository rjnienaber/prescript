package config

import "github.com/spf13/cobra"

type RecordConfig struct {
	DontCompress   bool
	ScriptFile     string
	ExecutablePath string
	Arguments      []string
}

func createRecordSubCommand(config *Config) *cobra.Command {
	var recordCmd = &cobra.Command{
		Use:   "record [script file] [executable] [flags] -- [args]",
		Short: "Runs a cli app and records output and responses",
		Long:  "Runs a cli app and records output and responses",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			config.Subcommand = RecordCommand
			config.Record.ScriptFile = args[0]
			config.Record.ExecutablePath = args[1]
			config.Record.Arguments = args[2:]
		},
	}

	recordCmd.Flags().BoolVarP(&config.Record.DontCompress, "dont-compress", "d", false, "don't compress lines to match on")

	return recordCmd
}
