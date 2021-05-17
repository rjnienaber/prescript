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
		lib.ProcessError(err, config.Logger, "script file couldn't be parsed")
		os.Exit(lib.RunPlay(config, script.Runs[0]))
	}

	//rollStep := lib.Step{
	//	Line:  ``,
	//	Input: "5000",
	//}
	//
	//tryAgainStep := lib.Step{
	//	Line:  ``,
	//	Input: "N",
	//}

	//params := lib.RunParameters{
	//	AppFilePath: os.Args[1],
	//	Args:        os.Args[2:],
	//	Logger:      zapLogger,
	//	Steps:       []lib.Step{rollStep, tryAgainStep},
	//}
	//exitCode := lib.Play(params)
	//os.Exit(exitCode)
}
