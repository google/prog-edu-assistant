# Docker files

The currently supported hosted deployment scenario is Google Cloud Run.

## Running the image locally with Docker

To build the docker containers, use the command

    ./build.sh

To run the combined service, use the following Docker command:

    docker run -p 8000:8000/tcp --rm --name combined combined:latest

The server should then be available on

    http://localhost:8000/
