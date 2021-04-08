package lib

import (
	"go.uber.org/zap"
	"time"
)

type Step struct {
	Line  string
	Input string
}

// TODO: merge struct for <script.json> with RunParameters
// https://stackoverflow.com/a/40510391/9539
type RunParameters struct {
	AppFilePath           string
	Args                  []string
	TimeoutInMilliseconds int64
	Logger                *zap.Logger
	Steps                 []Step
	ExitCode              int
}

func (params *RunParameters) Timeout() time.Duration {
	if params.TimeoutInMilliseconds != 0 {
		return time.Duration(params.TimeoutInMilliseconds) * time.Millisecond
	}
	return 30 * time.Millisecond
}
