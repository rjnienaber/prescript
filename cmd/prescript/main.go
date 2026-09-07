package main

import (
	"fmt"
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
	if err != nil {
		os.Exit(utils.USER_ERROR)
	}

	if config.Handled {
		os.Exit(utils.SUCCESS)
	}

	if config.Subcommand == cfg.NoCommand {
		os.Exit(utils.USER_ERROR)
	}

	level := "none"
	if config.Subcommand == cfg.PlayCommand {
		level = strings.ToLower(config.Play.LogLevel)
	}

	logger, err := utils.NewLogger(level)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(utils.USER_ERROR)
	}
	logger.Info("successfully parsed arguments and flags")
	config.Logger = &logger

	if config.Subcommand == cfg.PlayCommand {
		scriptFile, err := script.ParseScriptFromFile(config.Play.ScriptFile)
		if err != nil {
			// Also on stderr: logging alone is invisible at the default level,
			// so a malformed script file exited non-zero in silence.
			fmt.Fprintf(os.Stderr, "could not parse script file %s: %s\n", config.Play.ScriptFile, err)
			logger.Info("scriptFile file couldn't be parsed:", err)
			os.Exit(utils.USER_ERROR)
		}

		runs := scriptFile.Runs
		if config.Play.RunnerFile != "" {
			runner, err := script.ParseRunnerFromFile(config.Play.RunnerFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not parse runner file %s: %s\n", config.Play.RunnerFile, err)
				os.Exit(utils.USER_ERROR)
			}
			if config.Play.ExecutablePath != "" {
				fmt.Fprintf(os.Stderr, "warning: --exec overrides the executable named by runner %s\n", runner.Name)
			}
			logger.Infof("running with runner %s", runner.Name)
			runs = script.ApplyRunner(runs, runner)
		}

		result := play.RunAll(config.Play, runs, config.Logger)
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
