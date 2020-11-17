#!/usr/bin/env python

import copy
import json
import re

from absl import app
from absl import flags
from absl import logging

FLAGS = flags.FLAGS

flags.DEFINE_string('master_notebook', None,
        'The path to the master notebook file (.ipynb) to convert.')
flags.DEFINE_string('output_student_notebook', None,
        'The output path to write the converted student notebook. '
        'If not specified, the converted notebook is printed to stdout.')

def LoadNotebook(filename):
    """Load an ipynb notebook.

    Args:
        filename: the name of the .ipynb file.

    Returns
        loaded notbook as a JSON object.
    """
    with open(filename) as f:
        return json.load(f)

def SaveNotebook(notebook, filename):
    """Save a notebook to .ipynb file.

    Args:
        notebook: a notebook in the form of a JSON object.
        filename: the name of the .ipynb file to write.
    """
    with open(filename, 'w') as f:
        json.dump(notebook, f)


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


# A regexp identifying master-only notebooks. Applies both to code and markdown cells.
reMasterOnly = re.compile('^[\t ]*#.*MASTER ONLY.*$', re.M)
reTest = re.compile('^%%inlinetest')
reSubmission = re.compile('^%%submission[ \t]*\n')
reAutotest = re.compile('%autotest|autotest\\(')
reReport = re.compile('%%(template|report)|report\\(')
reSolution = re.compile('^%%solution[ \t]*\n')
reExerciseID = re.compile('^# *EXERCISE_ID: [\'"]?([a-zA-Z0-9_.-]*)[\'"]?\n', re.M)
reSolutionBegin = re.compile('^([ \t]*)# BEGIN SOLUTION[ \t]*\n', re.M)
reSolutionEnd = re.compile('^[ \t]*# END SOLUTION[ \t]*\n', re.M)
rePromptBegin = re.compile('^[ \t]*""" # BEGIN PROMPT[ \t]*\n', re.M)
rePromptEnd = re.compile('^[ \t]*""" # END PROMPT[ \t]*\n?', re.M)


def ShouldSkipCodeCell(source):
    """Returns true iff the cell should be skipped from student notebook.

    Args:
        source: The merged source string of the code cell.

    Returns:
        true iff the cell should be skipped in the student notebook output.
    """
    return (reMasterOnly.search(source) or
            reSubmission.search(source) or
            reAutotest.search(source) or
            reReport.search(source))

def ExtractPrompt(source, default):
    """Attempts to extract the prompt from the code cell.

    Args:
        source: The merged source string of the code cell.
        default: The default prompt string.

    Returns:
        The source with prompt removed.
        The first extracted prompt if prompt regexp matched, or default otherwise.
    """
    promptBeginMatch = rePromptBegin.search(source)
    promptEndMatch = rePromptEnd.search(source)
    if promptBeginMatch and promptEndMatch:
        if promptBeginMatch.end(0) > promptEndMatch.start(0):
            logging.error("Malformed prompt in cell:\n%s", source)
            return source, default
        return (source[:promptBeginMatch.start(0)] + source[promptEndMatch.end(0):],
            source[promptBeginMatch.end(0):promptEndMatch.start(0)])
    elif promptBeginMatch or promptEndMatch:
        logging.error("Malformed prompt in cell:\n%s", source)
    return source, default


def CleanCodeCell(source):
    """Rewrites the code cell source by removing markers.

    Args:
        source: The merged source string of the code cell.

    Returns
        A cleaned up source string.
    """
    m = reSolution.search(source)
    if m:
        source = source[m.end(0):]
    m = reExerciseID.search(source)
    if m:
        source = source[0:m.start(0)] + source[m.end(0):]
    m = reSolutionBegin.search(source)
    if m:
        indent = m.group(1)
        prompt = indent + '...\n'
        source, prompt = ExtractPrompt(source, prompt)
        outs = []
        while m:
            outs.append(source[0:m.start(0)])
            post = source[m.start(0):]
            m = reSolutionEnd.search(post)
            if not m:
                logging.error('Unclosed # SOLUTION BEGIN in cell:\n%s', source)
                outs.append(post)
                break
            outs.append(prompt)
            source = post[m.end(0):]
            # Update the prompt from the remaining piece.
            source, prompt = ExtractPrompt(source, prompt)
            m = reSolutionBegin.search(source)
            # Update the prompt from the remaining piece.
            source, prompt = ExtractPrompt(source, prompt)
        # Append the last remaining part.
        outs.append(source)
        source = ''.join(outs)
    return source


def ExtractExerciseID(source):
    """Attempts to extracts exercise ID from the code cell.

    Args:
        source: The merged source string of the code cell.

    Returns:
        exercide ID string if found, or None if not found.
    """
    m = reExerciseID.search(source)
    if m:
        return m.group(1)
    return None


def ToStudent(notebook):
    """Convert a master notebook to student notebook.

    It removes the cells that are recognized as tests and master-only cells,
    and removes some markers (e.g. # EXERCISE_ID) from the code.
    See the source code definition for the details of the transformations.

    Args:
        notebook: a master notebook in the form of a JSON object.

    Returns
        A converted student notebook in the form of a JSON object.
    """
    output_cells = []
    for cell in notebook['cells']:
        source = ''.join(cell['source'])
        if reMasterOnly.search(source):
            continue
        cell_type = cell['cell_type']
        if cell_type != 'code':
            output_cells.append(cell)
            continue
        # cell_type == 'code'
        if ShouldSkipCodeCell(source):
            continue
        source = CleanCodeCell(source)
        output_cell = {
            'cell_type': 'code',
            'source': source.splitlines(keepends=True),
        }
        output_cells.append(output_cell)
    output_notebook = copy.deepcopy(notebook)
    output_notebook['cells'] = output_cells
    return output_notebook


def main(argv):
    if not FLAGS.master_notebook:
        if len(argv) != 2:
            raise app.UsageError(f'Usage: convert_to_student.py <notebook file>')
        master_notebook_filename = argv[1]
    else:
        if len(argv) != 1:
            raise app.UsageError(f'Usage: convert_to_student.py --master_notebook <notebook file>')
        master_notebook_filename = FLAGS.master_notebook
    master_notebook = LoadNotebook(master_notebook_filename)
    student_notebook = ToStudent(master_notebook)
    if FLAGS.output_student_notebook:
        SaveNotebook(student_notebook, FLAGS.output_student_notebook)
    else:
        PrintNotebook(student_notebook)


if __name__ == '__main__':
    app.run(main)
