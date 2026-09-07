#!/bin/sh

# Reports whether its stdout is a terminal, which is the question a real
# program asks before deciding how to buffer its output.

if [ -t 1 ]; then
  echo "STDOUT=terminal"
else
  echo "STDOUT=pipe"
fi

printf 'Name: '
read -r name
echo "Hello, $name"
