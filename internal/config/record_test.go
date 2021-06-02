package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsesRecordCommand(t *testing.T) {
	config, err := testParseArgs([]string{"record", "/tmp/script.json", "/bin/ls"})
	assert.NoError(t, err)
	assert.Equal(t, RecordCommand, config.Subcommand)
	assert.Equal(t, "/tmp/script.json", config.Record.ScriptFile)
	assert.Equal(t, "/bin/ls", config.Record.ExecutablePath)
	assert.Equal(t, []string{}, config.Record.Arguments)
}

func TestParsesRecordExecutableArguments(t *testing.T) {
	config, err := testParseArgs([]string{"record", "/tmp/script.json", "/bin/ls", "--", "-r", "-l"})
	assert.NoError(t, err)
	assert.Equal(t, RecordCommand, config.Subcommand)
	assert.Equal(t, "/tmp/script.json", config.Record.ScriptFile)
	assert.Equal(t, "/bin/ls", config.Record.ExecutablePath)
	assert.Equal(t, []string{"-r", "-l"}, config.Record.Arguments)
}

func TestParsesRecordExecutableArguments2(t *testing.T) {
	config, err := testParseArgs([]string{"record", "/tmp/script.json", "/bin/ls", "--", "record", "--ignoreOutput"})
	assert.NoError(t, err)
	assert.Equal(t, RecordCommand, config.Subcommand)
	assert.Equal(t, "/tmp/script.json", config.Record.ScriptFile)
	assert.Equal(t, "/bin/ls", config.Record.ExecutablePath)
	assert.Equal(t, []string{"record", "--ignoreOutput"}, config.Record.Arguments)
}

func TestConfigDontCompress(t *testing.T) {
	config, err := testParseArgs([]string{"record", "/tmp/script.json", "/bin/ls", "--dont-compress"})
	assert.NoError(t, err)
	assert.Equal(t, RecordCommand, config.Subcommand)
	assert.Equal(t, "/tmp/script.json", config.Record.ScriptFile)
	assert.Equal(t, "/bin/ls", config.Record.ExecutablePath)
	assert.Equal(t, []string{}, config.Record.Arguments)
	assert.True(t, config.Record.DontCompress)
}
