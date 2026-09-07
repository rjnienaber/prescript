#!/bin/sh
#
# Prints the size the kernel reports for the terminal on stdin, which is what
# a program consults before deciding where to wrap its output.
stty size
