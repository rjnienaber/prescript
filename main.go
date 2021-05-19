package main

import (
	"os"
	"prescript/lib"
	"strings"
)

func main() {
	config, err := lib.GetConfig()
	if err != nil || config.Subcommand == lib.NoCommand {
		os.Exit(lib.USER_ERROR)
	}

	level := "none"
	if config.Subcommand == lib.PlayCommand {
		level = strings.ToLower(config.Play.LogLevel)
	}

	logger, err := lib.NewLogger(level)
	if err != nil {
		os.Exit(lib.INTERNAL_ERROR)
	}
	defer logger.Close()
	logger.Debug("successfully parsed arguments and flags")
	config.Logger = logger

	if config.Subcommand == lib.PlayCommand {
		script, err := lib.NewScriptFromFile(config.Play.ScriptFile)
		if err != nil {
			logger.Debug("script file couldn't be parsed:", err)
			os.Exit(lib.USER_ERROR)
		}
		os.Exit(lib.RunPlay(config, script.Runs[0]))
	}
}
