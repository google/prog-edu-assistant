# Deployment instructions

The recommended way to deploy autograder backend is Google Cloud Run.

## Prerequisites

One needs a few things set up in order to have an instance of the autochecking
server:

*   To configure a system properly, you need to know the URL of the Cloud Run
    instance. The easiest way is to deploy first image with incomplete
    configuration (the autograder will not work), then copy the service URL
    from Cloud Console page of the service. Since the URL is stable, you can
    now complete the configuration and redeploy the service.
    The below instruction uses `$GCE_HOST` to refer to this name.

*   An OAuth client ID pair (client ID and client Secret), obtainable from
    http://console.cloud.google.com under the section APIs & Services,
    subsection Credentials.

    The client ID must have `$GCE_HOST` in the list of authorized
    Javascript origins, and should have the URL of the form
    `https://$GCE_HOST/callback` in the list of authorized redirect URLs. It
    is also helpful to include `http://localhost:8000` and
    `http://localhost:8000/callback` respectively for local testing.

    The client ID and client secret should be stored in the file
    `deploy/cloud-run.env`. The the example `deploy/cloud-run.env.template`
    for the format. The environment file should also have `SERVER_URL` to be
    set to the `https://$GCE_HOST`, using the domain name obtained above.

WARNING: You should never submit secrets, certificats or private keys to
source code repository.

## Initial gcloud authentication (on a dev machine)

You need to install recent version of Google Cloud SDK first.

    # Authenticate with gcloud
    gcloud auth login
    # Choose the project name
    gcloud config set project ${GCP_PROJECT?}

Your project unique identifier is referred with `$GCP_PROJECT` below.

## Build and push images to GCR (on a dev machine)

You only need to run this step if you have made changes to the source code
base.

    (cd docker && ./build.sh && \
     docker tag server asia.gcr.io/${GCP_PROJECT?}/combined && \
     docker push asia.gcr.io/${GCP_PROJECT?}/combined)

Note that the file `deploy-cloud-run.sh` contains these and below commands
and can be used for convenience.

## Run a test instance (on a dev machine)

There are two shell scripts provided for starting server locally
for quick debugging:

  * `start-server-no-queue.sh` starts the binary using `go` command.
    It expects the autograder directory to be prepared in `tmp/autograder`.
    You can use the script `build-student.sh` to prepare the autograder
    directory. It expects the environment to be configured in
    `deploy/local.env`.

  * `start-local-combined.sh` starts the docker container that is built
    by `docker/build.sh`.

# Google Cloud Run deployment

Google Cloud Run is a an attractive deployment option, because it
provides automatic scaling from zero, which means that there is no
need for manual capacity planning. The autograding server works in
a combined model with the python grading working on the same
machine as the upload server.

Here is an example of the deploy command:

    docker/build.sh && \
    docker tag combined asia.gcr.io/${GCP_PROJECT?}/combined && \
    docker push asia.gcr.io/${GCP_PROJECT?}/combined && \
    @execute gcloud run deploy \
      combined \
      --image asia.gcr.io/${GCP_PROJECT?}/combined \
      --allow-unauthenticated \
      --platform=managed \
      --region asia-northeast1 \
      --set-env-vars=GCP_PROJECT="${GCP_PROJECT?}",\
    CLIENT_SECRET="$CLIENT_SECRET",\
    CLIENT_ID="$CLIENT_ID",\
    LOG_BUCKET="$LOG_BUCKET",\
    SERVER_URL="$SERVER_URL",\
    HASH_SALT="$HASH_SALT",\
    COOKIE_AUTH_KEY="$COOKIE_AUTH_KEY",\
    COOKIE_ENCRYPT_KEY="$COOKIE_ENCRYPT_KEY",\
    JWT_KEY="$JWT_KEY"

The environment variables normally should be stored in the file
`deploy/cloud-run.env`, which should not be submitted to Git.
Copy `deploy/cloud-run.env.template` and edit to fill the details.
Note that the file `deploy-cloud-run.sh` contains these commands
and can be used for convenience.

TODO(salikh): Provide an example command for generating the JWT key.
