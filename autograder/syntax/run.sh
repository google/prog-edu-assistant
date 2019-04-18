#!/bin/sh
#
# Runs the tests in the directory where the script is located.
# Usage:
#
#   ./run.sh
#

/usr/bin/env python3 -m unittest discover -v -s "$(dirname "$0")" -p '*_test.py'
