#!/bin/bash
#
# A convenience script to build and start a local docker instance with the
# combined server.

set -ex
cd "$(dirname "$0")"
bazel test ...
cd docker
./build.sh
docker run -p 8000:8000/tcp --rm --name combined combined:latest
