package lib

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func testParseArgs(args []string) (*Config, error) {
	newArgs := append([]string{"prescript"}, args...)
	os.Args = newArgs
	return GetConfig()
}

func TestConfigPlayQuiet(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json", "--quiet"})
	assert.NoError(t, err)
	assert.Equal(t, config.Subcommand, Play)
	assert.True(t, config.Play.Quiet)
}

func TestConfigDontFail(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json", "--dont-fail"})
	assert.NoError(t, err)
	assert.Equal(t, config.Subcommand, Play)
	assert.True(t, config.Play.DontFail)
}

func TestConfigVerbose(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json", "--verbose"})
	assert.NoError(t, err)
	assert.Equal(t, config.Subcommand, Play)
	assert.True(t, config.Play.Verbose)
}

func TestConfigTimeout(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json", "--timeout=10s"})
	assert.NoError(t, err)
	assert.Equal(t, config.Subcommand, Play)
	assert.Equal(t, config.Play.Timeout.Seconds(), 10.0)
	assert.Equal(t, config.Play.ScriptFile, "/tmp/script.json")
}

func TestConfigScriptFile(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json"})
	assert.NoError(t, err)
	assert.Equal(t, config.Subcommand, Play)
	assert.Equal(t, config.Play.ScriptFile, "/tmp/script.json")
}

func TestConfigAcceptsOptionalExecutable(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json", "/bin/ls"})
	assert.NoError(t, err)
	assert.Equal(t, config.Subcommand, Play)
	assert.Equal(t, config.Play.ScriptFile, "/tmp/script.json")
	assert.Equal(t, config.Play.Command, "/bin/ls")
}

func TestConfigRequiresScriptfile(t *testing.T) {
	_, err := testParseArgs([]string{"play"})
	assert.Errorf(t, err, "accepts between 1 and 2 arg(s), received 0")
}

func TestParsesRecordCommand(t *testing.T) {
	t.Skip("not implemented")
	config, _ := testParseArgs([]string{"record"})
	assert.Equal(t, config.Subcommand, Record)
}

func TestNoArgs(t *testing.T) {
	t.Skip("not implemented")
}

func TestFormatsHelp(t *testing.T) {
	t.Skip("not implemented")
}
