package main

import (
	"os"
	"prescript/lib"
)

func main() {
	config, err := lib.GetConfig()
	if err != nil || config.Subcommand == lib.NotSpecified {
		os.Exit(lib.INTERNAL_ERROR)
	}

	config.Logger.Debug("successfully parsed arguments and flags")

	//zapLogger := lib.CreateLogger()
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
