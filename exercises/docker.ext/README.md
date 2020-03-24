# Skeleton docker directory for external repository

This directory is intended as a skeleton Docker
directory for external repositories that need
to build deployment images for autograder backend.

## How to use

Copy the contents of this directory into a new external repository.

    cp -r exercises .../my-new-repo
    cd .../my-new-repo
    mv WORKSPACE.ext WORKSPACE
    mv BUILD.ext BUILD.bazel
    mv docker.ext docker

    bazel build ...
    ./docker/build-docker-image.sh
