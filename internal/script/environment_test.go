package script

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var parentEnv = []string{"HOME=/home/richard", "GREETING=inherited"}

func TestNoEnvMeansInherit(t *testing.T) {
	run := Run{}
	// nil, not an empty slice: exec reads nil as "inherit" and an empty slice
	// as "run with nothing at all", and the two must not be confused.
	assert.Nil(t, run.Environment(parentEnv))
}

func TestDeclaredEnvReplaces(t *testing.T) {
	run := Run{Env: map[string]string{"GREETING": "declared"}}
	assert.Equal(t, []string{"GREETING=declared"}, run.Environment(parentEnv))
}

func TestEmptyEnvIsAnEmptyEnvironment(t *testing.T) {
	run := Run{Env: map[string]string{}}
	assert.Equal(t, []string{}, run.Environment(parentEnv))
}

func TestInheritEnvPutsTheParentUnderneath(t *testing.T) {
	run := Run{Env: map[string]string{"GREETING": "declared"}, InheritEnv: true}
	// The declared value comes last, which is how exec resolves a name that
	// appears twice.
	assert.Equal(t, []string{"HOME=/home/richard", "GREETING=inherited", "GREETING=declared"}, run.Environment(parentEnv))
}

func TestInheritEnvWithoutEnvStillInherits(t *testing.T) {
	run := Run{InheritEnv: true}
	assert.Nil(t, run.Environment(parentEnv))
}

func TestEnvIsBuiltInAStableOrder(t *testing.T) {
	run := Run{Env: map[string]string{"C": "3", "A": "1", "B": "2"}}
	assert.Equal(t, []string{"A=1", "B=2", "C=3"}, run.Environment(nil))
}
