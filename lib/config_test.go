package lib

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func testParseArgs(args []string) *Config {
	newArgs := append([]string{"prescript"}, args...)
	os.Args = newArgs
	config := GetConfig()
	return config
}

func TestConfigPlayQuiet(t *testing.T) {
	config := testParseArgs([]string{"play", "--quiet"})
	assert.Equal(t, config.Subcommand, Play)
	assert.True(t, config.Play.Quiet)
}

func TestConfigDontFail(t *testing.T) {
	config := testParseArgs([]string{"play", "--dont-fail"})
	assert.Equal(t, config.Subcommand, Play)
	assert.True(t, config.Play.DontFail)
}

func TestConfigVerbose(t *testing.T) {
	config := testParseArgs([]string{"play", "--verbose"})
	assert.Equal(t, config.Subcommand, Play)
	assert.True(t, config.Play.Verbose)
}

func TestConfigTimeout(t *testing.T) {
	config := testParseArgs([]string{"play", "--timeout=10s"})
	assert.Equal(t, config.Subcommand, Play)
	assert.Equal(t, config.Play.Timeout.Seconds(), 10.0)
}

func TestParsesRecordCommand(t *testing.T) {
	t.Skip("not implemented")
	config := testParseArgs([]string{"record"})
	assert.Equal(t, config.Subcommand, Record)
}

func TestNoArgs(t *testing.T) {
	t.Skip("not implemented")
}

func TestFormatsHelp(t *testing.T) {
	t.Skip("not implemented")
}
