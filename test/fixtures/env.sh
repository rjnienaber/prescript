#!/bin/sh

# /bin/sh, not /usr/bin/env bash: this fixture is started with a replaced
# environment, and resolving the interpreter should not depend on whatever
# fallback PATH the platform substitutes when the variable is unset.

echo "GREETING=${GREETING:-unset}"
echo "PARENT=${PRESCRIPT_TEST_PARENT:-unset}"
