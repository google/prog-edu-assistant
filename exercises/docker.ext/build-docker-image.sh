#!/bin/bash
#
# Build the Docker images of autograder.
#
# Usage:
#
#  ./build-docker-image.sh
#

function @execute() { echo "$@" >&2; "$@"; }

DIR="$(dirname "$0")"
DIR="$(cd "$DIR"; pwd -P)"
STAGE="$DIR/stage"

set -e

@execute cd "$DIR/../"
@execute bazel build ...
@execute rm -rf "$STAGE/bin"
@execute rm -rf "$STAGE/autograder"
@execute rm -rf "$STAGE/static"
@execute tar xvfi bazel-bin/autograder_tar.tar -C "$STAGE"

@execute cd "$DIR"
@execute docker build stage -f stage/Dockerfile -t combined
