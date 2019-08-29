# Google Compute Engine (GCE)

## Prerequisites

One needs a few things set up in order to have an instance of the autochecking
server:

*   A domain name that is under your control. It should point to the static IP
    address allocated for the GCE instance. The below code uses `$GCE_HOST` to
    refer to this name.

*   An OAuth client ID pair (client ID and client Secret), obtainable from
    http://console.cloud.google.com under the section APIs & Services,
    subsection Credentials.

    The client ID must have the above domain name in the list of authorized
    Javascript origins,j and should have the URL of the form
    `https://$GCE_HOST/callback` in the list of authorized redirect URLs. It is
    also helpful to include `http://localhost:8000` and
    `http://localhost:8000/callback` respectively for local testing.

    The client ID and client secret should be stored in the file
    `deploy/secret.env`. The the example `deploy/secret.env.template` for the
    format. The environment file should also have `SERVER_URL` to be set to the
    `https://$GCE_HOST`, using the domain name chosen above.

*   An SSL cert and private key pair issued for OU equal to the chosen domain
    name above. The below instructions assume that they are copied into the
    workspace at `deploy/certs/privkey1.pem` and `deploy/certs/cert1.pem`. The
    easiest way to get a certificate is from Letsencrypt.

*   A service account key in JSON format should be downloaded from GCP console
    in advance and put into `deploy/service-account.json`. The corresponding GCP
    project name is referred as $GCP_PROJECT below.

WARNING: You should never submit secrets, certificats or private keys to source
code repository.

## Initial gcloud authentication (on a dev machine)

You need to install recent version of Google Cloud SDK first.

    # Authenticate with gcloud
    gcloud auth login
    # Choose the project name
    gcloud config set project ${GCP_PROJECT?}

## Build and push images to GCR (on a dev machine)

You only need to run this step if you have made changes to the source code base.

    (cd docker && ./build.sh && \
     docker tag server asia.gcr.io/${GCP_PROJECT?}/server && \
     docker tag worker asia.gcr.io/${GCP_PROJECT?}/worker && \
     docker push asia.gcr.io/${GCP_PROJECT?}/server && \
     docker push asia.gcr.io/${GCP_PROJECT?}/worker)

## Create an instance (on a dev machine)

    gcloud compute instances create prog-edu-assistant \
      --zone=asia-northeast1-b \
      --machine-type=n1-standard-1 \
      --image-family=cos-stable \
      --image-project=cos-cloud \
      --tags=http-server,https-server

    # See the IP address of the instance:
    gcloud compute instances list

Note: secret.env has two items that depend on the stable server address:

(1) `SERVER_URL` should contain the URL of the server, starting with http:// and
having the port, but without the final slash. Obviously the stable URL of the
server should resolve to the actual IP address of the instance. It is a good
idea to configure instance with a static IP address.

(2) The `CLIENT_ID` and `CLIENT_SECRET` used for OpenID Connect authentication
must list the domain of the server as an authorized domain, as well as have the
URL http://server:port/upload in the authorized redirect URI list.

The file `service-account.json` should be obtained from GCP console as a service
account key. You may need to edit docker-compose.yml file for your needs (e.g.
CORS origin or the names of cert and private key files).

    # Copy the deployment files to the instance:
    scp -r deploy/{certs,docker-compose.yml,secret.env,service-account.json} \
      $GCE_HOST:

## Start the autochecker server (on a GCE instance)

Start with logging to console:

    ssh $GCE_HOST
    mkdir -p logs
    cat service-account.json | docker login -u _json_key --password-stdin https://asia.gcr.io
    docker pull asia.gcr.io/${GCP_PROJECT?}/worker
    docker pull asia.gcr.io/${GCP_PROJECT?}/server
    docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v $PWD:$PWD -w=$PWD --entrypoint=sh docker/compose:1.24.0 -c "cat service-account.json | docker login -u _json_key --password-stdin https://asia.gcr.io && export GCP_PROJECT=${GCP_PROJECT?} && docker-compose up --scale worker=4"

Or start and detach (on a dev machine):

    ssh $GCE_HOST "mkdir -p logs && cat service-account.json | docker login -u _json_key --password-stdin https://asia.gcr.io && docker pull asia.gcr.io/${GCP_PROJECT?}/worker && docker pull asia.gcr.io/${GCP_PROJECT?}/server && docker run -d --rm -v /var/run/docker.sock:/var/run/docker.sock -v \$PWD:\$PWD -w=\$PWD --entrypoint=sh docker/compose:1.24.0 -c 'cat service-account.json | docker login -u _json_key --password-stdin https://asia.gcr.io && export GCP_PROJECT=${GCP_PROJECT?} && docker-compose up --scale worker=4'"

## Inspect running services on the GCE instance

    ssh $GCE_HOST
    docker ps

## Kill all services (without taking the GCE instance down)

    ssh $GCE_HOST
    docker ps -q | xargs -n1 docker kill

## Delete the instance after it is no longer needed (on a dev machine)

    gcloud compute instances delete prog-edu-assistant
