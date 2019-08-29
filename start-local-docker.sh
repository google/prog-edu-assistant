#!/bin/bash
#
# A convenience script to build and start a local docker instance with the
# all the containers that comprise the autochecker service (worker, server,
# rabbitmq).

set -ex
cd "$(dirname "$0")"
bazel test ...
cd docker
./build.sh
mkdir -p /tmp/autograder/scratch /tmp/autograder/uploads
docker-compose up
