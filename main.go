package main

import (
	"os"
	"strings"

	"prescript/lib"
	cfg "prescript/lib/config"
)

func main() {
	config, err := cfg.GetConfig()
	if err != nil || config.Subcommand == cfg.NoCommand {
		os.Exit(lib.USER_ERROR)
	}

	level := "none"
	if config.Subcommand == cfg.PlayCommand {
		level = strings.ToLower(config.Play.LogLevel)
	}

	logger, err := lib.NewLogger(level)
	if err != nil {
		os.Exit(lib.INTERNAL_ERROR)
	}
	defer logger.Close()
	logger.Debug("successfully parsed arguments and flags")
	config.Logger = &logger

	if config.Subcommand == cfg.PlayCommand {
		script, err := lib.NewScriptFromFile(config.Play.ScriptFile)
		if err != nil {
			logger.Debug("script file couldn't be parsed:", err)
			os.Exit(lib.USER_ERROR)
		}

		result := lib.RunPlay(config, script.Runs[0])
		if config.Play.DontFail {
			os.Exit(lib.SUCCESS)
		} else {
			os.Exit(result)
		}
	}
}
