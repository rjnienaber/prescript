package record

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	cfg "prescript/internal/config"
	script "prescript/internal/script"
	"prescript/internal/utils"
)

func captureIO(outputChannel chan utils.CapturedToken, inputChannel chan utils.CapturedToken) ([]utils.CapturedLine, int) {
	lines := []utils.CapturedLine{}
	currentInputLine := ""
	currentOutputLine := ""
	for {
		select {
		case outputToken := <-outputChannel:
			if outputToken.Error != nil {
				return nil, utils.CLI_ERROR
			}

			if outputToken.Finished {
				return lines, -1
			}

			char := outputToken.Token
			fmt.Print(char)
			if char == "\n" {
				lines = append(lines, utils.CapturedLine{Value: currentOutputLine, LineType: utils.Output})
				currentOutputLine = ""
				continue
			}

			currentOutputLine += char
		case inputToken := <-inputChannel:
			if inputToken.Error != nil {
				return nil, utils.INTERNAL_ERROR
			}

			if inputToken.Token == "\n" {
				if currentOutputLine != "" {
					lines = append(lines, utils.CapturedLine{Value: currentOutputLine, LineType: utils.Output})
					currentOutputLine = ""
				}

				lines = append(lines, utils.CapturedLine{Value: currentInputLine, LineType: utils.Input})
				currentInputLine = ""
			} else {
				currentInputLine += inputToken.Token
			}
		}
	}
}

func runRecord(config cfg.RecordConfig, logger utils.Logger) (string, int) {
	executable, err := utils.StartExecutable(config.ExecutablePath, config.Arguments, logger)
	if err != nil {
		return "", utils.INTERNAL_ERROR
	}

	outputChannel := createOutputChannel(executable.Stdout)
	inputChannel := createInputChannel(executable.Stdin, os.Stdin)

	lines, errorExitCode := captureIO(outputChannel, inputChannel)
	if errorExitCode != -1 {
		return "", errorExitCode
	}

	exitCode, err := executable.WaitForExit()
	if err != nil {
		return "", utils.INTERNAL_ERROR
	}

	scriptJson, err := script.BuildScriptJson(config, lines, exitCode, time.Now(), logger)
	if err != nil {
		return "", utils.INTERNAL_ERROR
	}

	return scriptJson, utils.SUCCESS
}

func Run(config cfg.RecordConfig, logger utils.Logger) int {
	scriptJson, exitCode := runRecord(config, logger)
	if exitCode == utils.SUCCESS {
		err := ioutil.WriteFile(config.ScriptFile, []byte(scriptJson), 0755)
		if err != nil {
			logger.Error("failed writing script json to file: ", err)
			return utils.INTERNAL_ERROR
		}
	}
	return exitCode
}
