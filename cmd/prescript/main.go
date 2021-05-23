package main

import (
	"os"
	"strings"

	cfg "github.com/rjnienaber/prescript/internal/config"
	"github.com/rjnienaber/prescript/internal/play"
	"github.com/rjnienaber/prescript/internal/record"
	"github.com/rjnienaber/prescript/internal/script"
	"github.com/rjnienaber/prescript/internal/utils"
)

func main() {
	config, err := cfg.GetConfig()
	if err != nil || config.Subcommand == cfg.NoCommand {
		os.Exit(utils.USER_ERROR)
	}

	level := "none"
	if config.Subcommand == cfg.PlayCommand {
		level = strings.ToLower(config.Play.LogLevel)
	}

	logger, err := utils.NewLogger(level)
	if err != nil {
		os.Exit(utils.INTERNAL_ERROR)
	}
	logger.Info("successfully parsed arguments and flags")
	config.Logger = &logger

	if config.Subcommand == cfg.PlayCommand {
		scriptFile, err := script.NewScriptFromFile(config.Play.ScriptFile)
		if err != nil {
			logger.Info("scriptFile file couldn't be parsed:", err)
			os.Exit(utils.USER_ERROR)
		}

		result := play.Run(config.Play, scriptFile.Runs[0], config.Logger)
		if config.Play.DontFail {
			os.Exit(utils.SUCCESS)
		} else {
			os.Exit(result)
		}
	}

	if config.Subcommand == cfg.RecordCommand {
		result := record.Run(config.Record, config.Logger)
		os.Exit(result)
	}
}
