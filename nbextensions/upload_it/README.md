# Upload it

This Jupyter notebook extension allows to upload the complete
Jupyter notebook to a configurable server. The primary intended
use case is for teaching programming to students, with the 
programming assignments distributed as Jupyter notebooks
with instructions and a few empty cells. Upon completion
of the assignment, the student can use this extension to
upload the complete notebook for automatic checking and grading.

The code was inspired by the Gist-it extension by Josh Barnes.

## Prerequisites

In order to use this notebook extension, you need the Jupyter notebook
installation. The good way to have without needing to change your system python
setup is to use virtualenv. The installation procedure for virtualenv differs
between platforms. For example, to install virtualenv on MacOSX with Homebrew:

    brew install python3        # Make sure python3 is installed.
    pip3 install virtualenv     # Install virtualenv.

On Debian GNU/Linux you need to use a different command to install virtualenv:

    apt-get install python-virtualenv  # Install virtualenv.

After that the setup procedure is common for all platforms:

    virtualenv -p python3 venv  # Create the virtual Python environment in venv/
    source ./venv/bin/activate  # Activate it.
    pip install jupyter         # Install Jupyter (inside of ./venv).


## Installation

For developing the extension, use these two commands
in the same environment that you have Jupyter installed:

    jupyter nbextension install /path/to/nbextensions/upload_it --symlink
    jupyter nbextension enable upload_it/main

## Student installation

For the student installation, use:

    pip install jupyter_nbextensions_configurator
    jupyter nbextension install /path/to/nbext/upload_it
    jupyter nbextension enable upload_it/main

If you need to change the upload URL, you can do that
in the Nbextesion tab by clicking on the title of the extension
("Upload it"), then by editing the upload URL in the parameters section
below.

## Usage

Start the Jupyter notebooks by the command

    jupyter notebook

Open the notebook and complete the solution in the provided empty cell. Then
find the "Upload it" toolbar button, which has a check icon and press it. A
dialog window should appear. Click on the "Upload" button to upload the
notebook.
