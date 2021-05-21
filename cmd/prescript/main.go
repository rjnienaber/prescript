package main

import (
	"os"
	"strings"

	"prescript/internal"
	cfg "prescript/internal/config"
)

func main() {
	config, err := cfg.GetConfig()
	if err != nil || config.Subcommand == cfg.NoCommand {
		os.Exit(internal.USER_ERROR)
	}

	level := "none"
	if config.Subcommand == cfg.PlayCommand {
		level = strings.ToLower(config.Play.LogLevel)
	}

	logger, err := internal.NewLogger(level)
	if err != nil {
		os.Exit(internal.INTERNAL_ERROR)
	}
	logger.Info("successfully parsed arguments and flags")
	config.Logger = &logger

	if config.Subcommand == cfg.PlayCommand {
		script, err := internal.NewScriptFromFile(config.Play.ScriptFile)
		if err != nil {
			logger.Info("script file couldn't be parsed:", err)
			os.Exit(internal.USER_ERROR)
		}

		result := internal.RunPlay(config, script.Runs[0])
		if config.Play.DontFail {
			os.Exit(internal.SUCCESS)
		} else {
			os.Exit(result)
		}
	}
}
