#!/bin/bash
#
# A convenience script to build and start a local docker instance with the
# autograder backend in the insecure mode, intended for local development
# and testing.

DIR="$(dirname "$0")"
DIR="$(cd "$DIR"; pwd -P)"
set -ex
cd "$DIR/.."
bazel build ...
cd "$DIR"
./build-docker-image.sh
# Note: the following flags are appended to ENTRYPOINT defined in the Dockerfile.
docker run -p 8000:8000/tcp --rm --name combined combined:latest \
  --logtostderr \
  --v=5 \
  --disable_cleanup \
  --auto_remove \
  --log_to_bucket=0 \
  --use_openid=0 \
  --use_jwt=0 \
  --secure_cookie=0
