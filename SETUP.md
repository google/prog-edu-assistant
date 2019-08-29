# Development environment setup

There are a few parts that are necessary for local development:

*   Virtualenv setup for Python and Jupyter for authoring master notebooks.
*   Docker (and Docker Compose) setup for local testing of autograder backend.
*   GCloud setup for deployment of autograder backend to GCE.

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

You will need to install Docker and Docker Compose.
