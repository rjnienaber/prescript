package lib

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func testParseArgs(args []string) (Config, error) {
	newArgs := append([]string{"prescript"}, args...)
	os.Args = newArgs
	return GetConfig()
}

func TestConfigPlayQuiet(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json", "--quiet"})
	assert.NoError(t, err)
	assert.Equal(t, Play, config.Subcommand)
	assert.True(t, config.Play.Quiet)
}

func TestConfigDontFail(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json", "--dont-fail"})
	assert.NoError(t, err)
	assert.Equal(t, Play, config.Subcommand)
	assert.True(t, config.Play.DontFail)
}

func TestConfigLogLevel(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json", "--log-level=info"})
	assert.NoError(t, err)
	assert.Equal(t, Play, config.Subcommand)
	assert.Equal(t, "info", config.Play.LogLevel)
}

func TestConfigTimeout(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json", "--timeout=10s"})
	assert.NoError(t, err)
	assert.Equal(t, Play, config.Subcommand)
	assert.Equal(t, 10.0, config.Play.Timeout.Seconds())
	assert.Equal(t, "/tmp/script.json", config.Play.ScriptFile)
}

func TestConfigScriptFile(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json"})
	assert.NoError(t, err)
	assert.Equal(t, Play, config.Subcommand)
	assert.Equal(t, "/tmp/script.json", config.Play.ScriptFile)
}

func TestConfigAcceptsOptionalExecutable(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json", "/bin/ls"})
	assert.NoError(t, err)
	assert.Equal(t, Play, config.Subcommand)
	assert.Equal(t, "/bin/ls", config.Play.ExecutablePath)
}

func TestConfigRequiresScriptfile(t *testing.T) {
	_, err := testParseArgs([]string{"play"})
	assert.Errorf(t, err, "accepts between 1 and 2 arg(s), received 0")
}

func TestParsesRecordCommand(t *testing.T) {
	config, err := testParseArgs([]string{"record", "/tmp/script.json", "/bin/ls"})
	assert.NoError(t, err)
	assert.Equal(t, Record, config.Subcommand)
	assert.Equal(t, "/tmp/script.json", config.Record.ScriptFile)
	assert.Equal(t, "/bin/ls", config.Record.ExecutablePath)
	assert.Equal(t, []string{}, config.Record.Arguments)
}

func TestParsesRecordIgnoreOutput(t *testing.T) {
	config, err := testParseArgs([]string{"record", "/tmp/script.json", "/bin/ls", "--ignoreOutput"})
	assert.NoError(t, err)
	assert.Equal(t, Record, config.Subcommand)
	assert.Equal(t, "/tmp/script.json", config.Record.ScriptFile)
	assert.Equal(t, "/bin/ls", config.Record.ExecutablePath)
	assert.True(t, config.Record.IgnoreOutput)
}

func TestParsesRecordExecutableArguments(t *testing.T) {
	config, err := testParseArgs([]string{"record", "/tmp/script.json", "/bin/ls", "--", "-r", "-l"})
	assert.NoError(t, err)
	assert.Equal(t, Record, config.Subcommand)
	assert.Equal(t, "/tmp/script.json", config.Record.ScriptFile)
	assert.Equal(t, "/bin/ls", config.Record.ExecutablePath)
	assert.Equal(t, []string{"-r", "-l"}, config.Record.Arguments)
}

func TestParsesRecordExecutableArguments2(t *testing.T) {
	config, err := testParseArgs([]string{"record", "/tmp/script.json", "/bin/ls", "--", "record", "--ignoreOutput"})
	assert.NoError(t, err)
	assert.Equal(t, Record, config.Subcommand)
	assert.Equal(t, "/tmp/script.json", config.Record.ScriptFile)
	assert.Equal(t, "/bin/ls", config.Record.ExecutablePath)
	assert.Equal(t, []string{"record", "--ignoreOutput"}, config.Record.Arguments)
}

func TestNoArgs(t *testing.T) {
	config, err := testParseArgs([]string{})
	assert.NoError(t, err)
	assert.Equal(t, NotSpecified, config.Subcommand)
}
