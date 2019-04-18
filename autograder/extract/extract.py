#!/usr/bin/env python3
"""A demo of extracting the exercise code from a submitted Jupyter notebook.
"""

from absl import app
from absl import flags
import ast
import json
import os
import re
import sys

FLAGS = flags.FLAGS
flags.DEFINE_string('input_file',
    os.path.join(os.path.dirname(__file__), 'submission.ipynb'),
    'The name of file with Jupyter notebook submission.')
flags.DEFINE_string('assignment_id', 'hello1',
    'The name of assignment to extract by matching against the cell metadata.')

def readMetadata(data, key):
    """Safely reads the value of data.metadata[key]."""
    if 'metadata' not in data:
        return None
    m = data['metadata']
    if key not in m:
        return None
    return m[key]


def main(argv):
  if len(argv) > 1:
    raise app.UsageError('Too many command-line arguments.')
  if FLAGS.input_file is None:
    raise app.UsageError('--input_file must be set.')
  try:
    with open(FLAGS.input_file) as f:
      src = f.read()
  except Exception as e:
    raise IOError("Could not open --input_file %s: %s" % (FLAGS.input_file, e))
  try:
    notebook = json.loads(src)
  except Exception as e:
    raise IOError("Could not parse --input_file as Jupyter notebook: %s" % e)
  # Note: more involved ways of detecting the solution cell are possible.
  # Some ideas:
  # - extract a cell that has a specific function definition;
  # - extract a code cell that follows a text cell matching a regexp.
  # Here we just assume the solution is in the last code cell.
  src = None
  for cell in notebook['cells']:
    # Skip all non-code cells.
    if cell['cell_type'] != 'code':
      continue
    if readMetadata(cell, 'assignment') == FLAGS.assignment_id:
      src = ''.join(cell['source'])

  if src is None:
    print("Could not find solution cell in the notebook, does it have a code cell with metadata?")
    sys.exit()

  print("The submitted code is\n---\n%s\n---" % src)

  try:
    tree = ast.parse(src)
  except Exception as e:
    print("ERROR The solution could not be parsed: %s" % e)
    sys.exit()

  regex1 = re.compile("print")
  if not regex1.search(src):
    print("ERROR The solution does not have print")
    sys.exit()

  regex2 = re.compile('"Hello')
  if not regex2.search(src):
    print("The solution does not have \"Hello\" string")
    sys.exit()

  print("The solution looks okay")


if __name__ == '__main__':
  app.run(main)
