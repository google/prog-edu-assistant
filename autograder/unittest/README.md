# Unittest

This directory contains a few examples of automated tests based on Python
unittest library. Each directory contains the following files:

*   `submission.py` --- this is where the extracted student's code will be
    written. Normally only the contents of one code cell will be written to the
    file.

*   `*_test.py` --- the tests are written normally using unittest or other
    python libraries. Normally the contents of the tests is produced from the
    test cells of the assignment master notebook.

This directory contains:

*   `run.sh` --- the command to run the tests under nsjail. Usage:

    run.sh <directories>

    This file is provided for demonstration purpose, as the actual command is
    likely to be hardcoded into the worker daemon.

In production deployments, the tests will be extracted from master notebooks
rather than from this directory.

## Prerequisites

### Install nsjail

On Debian-based Linux systems:

    apt-get install autoconf bison flex gcc g++ git libprotobuf-dev \
      libnl-route-3-dev libtool make pkg-config protobuf-compiler
    git clone https://github.com/google/nsjail.git
    (cd nsjail && make) && sudo cp nsjail/nsjail /usr/local/bin
