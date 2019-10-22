# Docker files

There is a couple ways how you can run the autograder backend.

## Separate workers and upload server with a message queue on Docker Compose

To build the docker containers of upload server and autograder worker, use the
command:

    ./build.sh

To run all services using Docker Compose:

    docker-compose up

## A combined upload and grading server on Cloud Run

To build the docker containers, use the command

    ./build.sh

To run the combined service, use the following Docker command:

    docker
