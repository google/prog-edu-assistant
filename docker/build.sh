#!/bin/bash
#
# Build Docker images of upload server and worker.
# Note: Message Queue uses stock RabbitMQ docker image.
#
# Usage:
#
#  ./build.sh
#  docker-compose up

function @execute() { echo "$@" >&2; "$@"; }

DIR="$(dirname "$0")"
DIR="$(cd "$DIR"; pwd -P)"
STAGE="$DIR/stage"

set -e

@execute cd "$DIR/../"
@execute bazel build ...
@execute rm -rf "$STAGE/autograder"
@execute tar xvf bazel-genfiles/exercises/autograder_image-layer.tar -C "$STAGE"
@execute rm -rf "$STAGE/bin" && mkdir "$STAGE/bin"
@execute cp -L "bazel-bin/go/cmd/worker/linux_amd64_stripped/worker" "$STAGE/bin"
@execute cp -L "bazel-bin/go/cmd/uploadserver/linux_amd64_stripped/uploadserver" "$STAGE/bin"
@execute rm -rf "$STAGE/static"
@execute cp -r "static" "$STAGE/"

@execute bazel run //go/cmd/uploadserver:docker -- --norun
@execute bazel run //go/cmd/worker:docker -- --norun

@execute cd "$DIR"
@execute docker build stage -f stage/Dockerfile.server -t server
@execute docker build stage -f stage/Dockerfile.worker -t worker
@execute docker build stage -f stage/Dockerfile.combined -t combined
