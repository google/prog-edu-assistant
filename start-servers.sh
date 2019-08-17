#!/bin/bash
#
# A debugging script to run the autograder worker daemon locally without
# docker. It assumes that the assets (autograder scripts) has been already
# generated in the directory ./tmp-autograder (use can use ./rebuild.sh).
# It also assumes that virtualenv installation is at ../venv.
#
# Usage:
#
#   ./rebuild.sh
#   ./start-servers.sh

cd "$(dirname "$0")"
source ../venv/bin/activate

set -ve

# Start Jupyter notebook server
pgrep jupyter &>/dev/null || jupyter notebook &

# Start message queue:
#sudo /etc/init.d/rabbitmq-server start

# Start the RabbitMQ using Docker.
docker run --rm -p 5672:5672 rabbitmq &

cd go
mkdir -p ../tmp-uploads
# Start the autograder worker
go run cmd/worker/worker.go --autograder_dir=../tmp-autograder --logtostderr --v=3 &

# Stop the processes we started on Ctrl+C
trap 'kill %3; kill %2; kill %1' SIGINT

# Start the upload server
go run cmd/uploadserver/main.go \
  --logtostderr --v=3 \
  --upload_dir=../tmp-uploads \
  --allow_cors_origin=http://localhost:8888
