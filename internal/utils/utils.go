package utils

const (
	SUCCESS = iota
	CLI_ERROR
	INTERNAL_ERROR
	USER_ERROR
)

type CapturedToken struct {
	Finished bool
	Token    string
	Error    error
}

type Logger interface {
	Close()
	Debug(args ...interface{})
	Debugf(template string, args ...interface{})
	Error(args ...interface{})
	Info(args ...interface{})
}

type lineType int

const (
	Input lineType = iota
	Output
)

type CapturedLine struct {
	Value    string
	LineType lineType
}
