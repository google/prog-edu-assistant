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

set -e

@execute cd "$DIR/../"
@execute bazel build ...
@execute rm -rf "$DIR/worker/autograder" && mkdir "$DIR/worker/autograder"
for i in bazel-genfiles/exercises/*-autograder; do
  @execute cp -rL "$i"/* "$DIR/worker/autograder/"
done
@execute rm -rf "$DIR/worker/bin" && mkdir "$DIR/worker/bin"
@execute cp -L "bazel-bin/go/cmd/worker/linux_amd64_stripped/worker" "$DIR/worker/bin"
@execute rm -rf "$DIR/server/bin" && mkdir "$DIR/server/bin"
@execute cp -L "bazel-bin/go/cmd/uploadserver/linux_amd64_stripped/uploadserver" "$DIR/server/bin"

@execute bazel run //go/cmd/uploadserver:docker -- --norun
@execute bazel run //go/cmd/worker:docker -- --norun

@execute cd "$DIR"
@execute docker build server -t server
@execute docker build worker -t worker
