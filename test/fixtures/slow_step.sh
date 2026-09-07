#!/bin/sh

# One slow moment in an otherwise quick program, which is the case a per-step
# timeout exists for: the script should not have to give every step the
# patience this one needs.

echo "Working"
sleep 2
echo "Done"
