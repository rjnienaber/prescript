package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigPlayQuiet(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json", "--quiet"})
	assert.NoError(t, err)
	assert.Equal(t, PlayCommand, config.Subcommand)
	assert.True(t, config.Play.Quiet)
}

func TestConfigDontFail(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json", "--dont-fail"})
	assert.NoError(t, err)
	assert.Equal(t, PlayCommand, config.Subcommand)
	assert.True(t, config.Play.DontFail)
}

func TestConfigLogLevel(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json", "--log-level=info"})
	assert.NoError(t, err)
	assert.Equal(t, PlayCommand, config.Subcommand)
	assert.Equal(t, "info", config.Play.LogLevel)
}

func TestConfigTimeout(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json", "--timeout=10s"})
	assert.NoError(t, err)
	assert.Equal(t, PlayCommand, config.Subcommand)
	assert.Equal(t, 10.0, config.Play.Timeout.Seconds())
	assert.Equal(t, "/tmp/script.json", config.Play.ScriptFile)
}

func TestConfigScriptFile(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json"})
	assert.NoError(t, err)
	assert.Equal(t, PlayCommand, config.Subcommand)
	assert.Equal(t, "/tmp/script.json", config.Play.ScriptFile)
}

func TestConfigAcceptsOptionalExecutable(t *testing.T) {
	config, err := testParseArgs([]string{"play", "/tmp/script.json", "/bin/ls"})
	assert.NoError(t, err)
	assert.Equal(t, PlayCommand, config.Subcommand)
	assert.Equal(t, "/bin/ls", config.Play.ExecutablePath)
}

func TestConfigRequiresScriptfile(t *testing.T) {
	_, err := testParseArgs([]string{"play"})
	assert.Errorf(t, err, "accepts between 1 and 2 arg(s), received 0")
}
