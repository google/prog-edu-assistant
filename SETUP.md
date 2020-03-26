# Development environment setup

This project is mainly developed using Linux (Debian or derivative).
It is possible that authoring workflow could work on MacOS as well,
but we are not testing it.

The requirements are different depending on whether you are only
interested in develoment of the Python assignment notebooks (master notebooks)
or if you need to modify the server or autograder and build deployment
images.

## Setup for authoring new assignments (master notebooks)

Install virtualenv. The command may differ depending on the system.

    # On Debian or Ubuntu linux
    apt-get install python-virtualenv  # Install virtualenv.

    # On MacOS with Homebrew:
    brew install python3        # Make sure python3 is installed.
    pip3 install virtualenv     # Install virtualenv.

After that the setup procedure is common

    virtualenv -p python3 ../venv    # Create the virtual Python environment in ../venv/
    source ../venv/bin/activate      # Activate it.
    pip install -r requirements.txt  # Install Jupyter etc. into ../venv.

Install the `prog_edu_assistant_tools`, or see
`python/prog_edu_assistant_tools/README.md` for the details.

    cd python/prog_edu_assistant_tools/
    source ../../../venv/bin/activate
    python setup.py bdist_wheel
    pip install --ignore-installed dist/prog_edu_assistant_tools-0.1-py3-none-any.whl
    cd ../..

To start the Jupyter notebook run command

    jupyter notebook

## Setup for local autograder backend development

The local development requires the following tools:

*   Virtualenv setup for Python and Jupyter for authoring master notebooks, as
    described above.
*   Go toolchain (https://golang.org), because the server and autograder are
    implemented in Go.
*   Bazel (http://bazel.build).
*   Docker (https://docs.docker.com/install) is used for building deployment
    container images and local testing of autograder backend.
*   Google Cloud SDK (https://cloud.google.com/sdk/install) is used for
    deployment of autograder backend to GCE.
