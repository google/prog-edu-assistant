#!/bin/bash
#
# A debugging script to run the autograder worker daemon locally without
# docker and without a message queue.
# It assumes that the assets (autograder scripts) has been already
# generated in the directory ./tmp/autograder (use can use ./rebuild.sh).
# It also assumes that virtualenv installation is at ../venv.
#
# Usage:
#
#   ./build-student.sh
#   ./start-server-no-queue.sh

cd "$(dirname "$0")"
DIR="$(pwd -P)"
source ../venv/bin/activate

set -ex

# Start Jupyter notebook server
pgrep jupyter &>/dev/null || jupyter notebook &

cd go
mkdir -p "$DIR/tmp/uploads" "$DIR/tmp/scratch"

# Stop the processes we started on Ctrl+C
trap 'kill %1' SIGINT

. ../docker/secret.env
export COOKIE_AUTH_KEY COOKIE_ENCRYPT_KEY CLIENT_ID CLIENT_SECRET

# Start the upload server
go run cmd/uploadserver/main.go \
  --logtostderr --v=5 \
  --upload_dir="$DIR/tmp/uploads" \
  --allow_cors \
  --use_openid \
  --static_dir="$DIR/static" \
  --grade_locally \
  --autograder_dir="$DIR/tmp/autograder" \
  --scratch_dir="$DIR/tmp/scratch" \
  --python_path="$(which python)" \
  --nsjail_path="$(which nsjail)" \
  --disable_cleanup \
  --auto_remove \
