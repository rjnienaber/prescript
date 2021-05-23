package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testParseArgs(args []string) (Config, error) {
	newArgs := append([]string{"prescript"}, args...)
	os.Args = newArgs
	return GetConfig()
}

func TestNoArgs(t *testing.T) {
	config, err := testParseArgs([]string{})
	assert.NoError(t, err)
	assert.Equal(t, NoCommand, config.Subcommand)
}
