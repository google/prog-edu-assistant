#!/bin/sh
#
# Runs the tests in a directory under nsjail.
# Usage:
#
#   run.sh dir_with_test
#

if [ -z "$*" ]; then
  echo "Usage: run.sh <directory>" >&2
  exit 1
fi

for i in "$@"; do
  if [ ! -d "$i" ]; then continue; fi
  echo "== $i =============="
  nsjail \
    -Mo \
    --time_limit 2 \
    --max_cpus 1 \
    --rlimit_as 100 \
    -E LANG=en_US.UTF-8 \
    --disable_proc \
    --chroot / \
    --cwd $PWD \
    --user nobody \
    --group nogroup \
    --iface_no_lo \
  -- /usr/bin/python -m unittest discover -v -s "$i" -p '*_test.py'
done
