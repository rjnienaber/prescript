#!/bin/sh

# Hangs, and leaves a background process of its own hanging with it — an
# interpreter that started the program being tested looks like this.
#
# $1 is a path the background process creates shortly after prescript gives up
# on this one. Kill the direct child alone and the file appears; kill the
# process group and it never does.

echo "Started"

( sleep 1; echo alive > "$1" ) &

sleep 987654
