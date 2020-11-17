#!/usr/bin/env python

import json
import re

from absl import app
from absl import flags
from absl import logging

FLAGS = flags.FLAGS

def LoadNotebook(filename):
    """Load an ipynb notebook.

    Args:
        filename: the name of the .ipynb file.

    Returns
        loaded notbook as a JSON object.
    """
    with open(filename) as f:
        return json.load(f)


def PrintNotebook(notebook):
    """Convert a master notebook to student notebook.

    It removes the cells that are recognized as tests and master-only cells,
    and removes some markers (e.g. # EXERCISE_ID) from the code.
    See the source code definition for the details of the transformations.

    Args:
        notebook: a master notebook in the form of a JSON object.

    Returns
        A converted student notebook in the form of a JSON object.
    """
    for cell in notebook['cells']:
        source = ''.join(cell['source'])
        print('-- ' + cell['cell_type'])
        print(source)


def main(argv):
    if len(argv) == 1:
        raise app.UsageError(f'Usage: cat_notebook.py <notebook file>...')
    for filename in argv[1:]:
        notebook = LoadNotebook(filename)
        PrintNotebook(notebook)


if __name__ == '__main__':
    app.run(main)
