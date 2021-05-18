package main

import (
	"os"
	"prescript/lib"
)

func main() {
	zapLogger := lib.CreateLogger()
	logger := zapLogger.Sugar()
	// sync errors can be ignored: https://github.com/uber-go/zap/issues/328
	defer logger.Sync()

	config, err := lib.GetConfig()
	if err != nil || config.Subcommand == lib.NotSpecified {
		os.Exit(lib.USER_ERROR)
	}
	logger.Debug("successfully parsed arguments and flags")
	config.Logger = logger

	if config.Subcommand == lib.Play {
		script, err := lib.NewScriptFromFile(config.Play.ScriptFile)
		if err != nil {
			logger.Debug("script file couldn't be parsed:", err)
			os.Exit(lib.USER_ERROR)
		}
		os.Exit(lib.RunPlay(config, script.Runs[0]))
	}
}
