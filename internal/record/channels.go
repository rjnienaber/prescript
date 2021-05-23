package record

import (
	"bufio"
	"io"

	"prescript/internal/utils"
)

func createOutputChannel(stdout io.ReadCloser) chan utils.CapturedToken {
	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanRunes)

	outputChannel := make(chan utils.CapturedToken)
	go func() {
		for {
			result := scanner.Scan()
			if !result {
				outputChannel <- utils.CapturedToken{Finished: true, Token: scanner.Text()}
				break
			}

			outputChannel <- utils.CapturedToken{Token: scanner.Text()}
		}
	}()
	return outputChannel
}

func createInputChannel(executableStdin io.WriteCloser, stdin io.Reader) chan utils.CapturedToken {
	reader := bufio.NewReader(stdin)
	inputChannel := make(chan utils.CapturedToken)
	go func() {
		for {
			result, _, err := reader.ReadRune()
			if err != nil {
				inputChannel <- utils.CapturedToken{Error: err}
				break
			}
			char := string(result)
			inputChannel <- utils.CapturedToken{Token: char}
			_, err = executableStdin.Write([]byte(char))

			// putting the error handling here to avoid timing issues with the executable
			if err != nil {
				inputChannel <- utils.CapturedToken{Error: err}
				break
			}
		}
	}()
	return inputChannel
}
