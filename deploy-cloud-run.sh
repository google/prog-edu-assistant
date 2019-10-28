#!/bin/bash

function @execute() { echo "$@" >&2; "$@"; }

DIR="$(dirname "$0")"
set -e

if [ ! -f "$DIR/deploy/cloud-run.env" ]; then
  echo "Please copy deploy/secret.env.template " >&2
  echo "to deploy/cloud-run.env and customize it." >&2
  exit 1
fi

source "$DIR/deploy/cloud-run.env"

"$(dirname "$0")/docker/build.sh"
@execute docker tag combined asia.gcr.io/${GCP_PROJECT?}/combined
@execute docker push asia.gcr.io/${GCP_PROJECT?}/combined

@execute gcloud beta run deploy \
  combined \
  --image asia.gcr.io/${GCP_PROJECT?}/combined \
  --allow-unauthenticated \
  --platform=managed \
  --region asia-northeast1 \
  --set-env-vars=GCP_PROJECT=${GCP_PROJECT?},\
LOG_BUCKET=$LOG_BUCKET,\
SERVER_URL=$SERVER_URL,\
HASH_SALT=$HASH_SALT,\
COOKIE_AUTH_KEY=$COOKIE_AUTH_KEY,\
COOKIE_ENCRYPT_KEY=$COOKIE_ENCRYPT_KEY

echo OK
