#!/usr/bin/env python
#
# A tool to grade students submitted notebooks against the instructor notebook.
#
# Usage example:
#
#     grade_student.py \
#       --instructor_notebook instructor.ipynb \
#       --student_notebook student.ipynb \
#       --context 'import tensorflow as tf; import numpy as np;'
#

import copy
import json
import re

from absl import app
from absl import flags
from absl import logging

from prog_edu_assistant_tools.magics import report, autotest, CaptureOutput
import convert_to_student

FLAGS = flags.FLAGS

flags.DEFINE_string('instructor_notebook', None,
        'The path to the instructor notebook file to check against.')
flags.DEFINE_string('student_notebook', None,
        'The path to the student notebook to check.')
flags.DEFINE_string('context', None,
        'A context piece of code to run before exercises.')
flags.DEFINE_bool('verbose', False,
        'If true, print more information about grading, including submitted '
        'snippets.')


def main(argv):
    if not FLAGS.instructor_notebook or not FLAGS.student_notebook:
        raise app.UsageError(f'Usage: grade_student.py --instructor_notebook '
            '<notebook file> --student_notebook <notebook file>')
    instructor_notebook = convert_to_student.LoadNotebook(FLAGS.instructor_notebook)
    canonical_notebook = convert_to_student.ToStudent(instructor_notebook, embed_inline_tests=True)
    submission_notebook = convert_to_student.LoadNotebook(FLAGS.student_notebook)
    num_exercises = 0
    num_submitted = 0
    num_tests = 0
    num_passed = 0
    num_failed = 0
    num_errors = 0
    for cell in canonical_notebook['cells']:
        if not 'metadata' in cell or not  'exercise_id' in cell['metadata']:
            # Skip non-exercise cells.
            continue
        metadata = cell['metadata']
        exercise_id = metadata['exercise_id']
        empty_submission = ''.join(cell['source'])
        logging.info(f'# exercise_id: {exercise_id}')
        num_exercises += 1
        found = False
        for sub_cell in submission_notebook['cells']:
            if not 'metadata' in sub_cell or not  'exercise_id' in sub_cell['metadata']:
                # Skip non-exercise cells.
                continue
            if sub_cell['metadata']['exercise_id'] != exercise_id:
                continue
            found = True
            break
        if not found:
            continue
        if not 'source' in sub_cell:
            continue
        submission_source = ''.join(sub_cell["source"])
        if submission_source == empty_submission:
            continue
        num_submitted += 1
        if FLAGS.verbose:
            logging.info('# submission\n---\n' + submission_source + '\n---')
        if 'inlinetests' in metadata:
            for test_name, test_source in metadata['inlinetests'].items():
                logging.info(f'# test: {test_name}')
                num_tests += 1
                if FLAGS.verbose:
                    logging.info(f'# running inline test {test_name}')
                errors =  []
                with CaptureOutput() as (stdout, stderr):
                    try:
                        env = {}
                        if FLAGS.context is not None:
                            logging.info("--- Executing context:\n%s\n---", FLAGS.context)
                            exec(FLAGS.context, env)
                    except Exception as e:
                        num_errors += 1
                        if FLAGS.verbose:
                            logging.info('error: ' + str(e))
                        continue
                    try:
                        try:
                            logging.info("--- Executing submission source:\n%s\n---", submission_source)
                            exec(submission_source, env)
                        except Exception as e:
                            raise AssertionError(str(e))
                        logging.info("--- Executing test source:\n%s\n---", test_source)
                        try:
                            exec(test_source, env)
                        except Exception as e:
                            raise AssertionError(str(e))
                    except AssertionError as e:
                        errors.append(str(e))
                        if FLAGS.verbose:
                            logging.info(f'#failed with ' + str(e))
                    if len(stderr.getvalue()) > 0:
                        errors.append('STDERR: ' + stderr.getvalue())
                        if FLAGS.verbose:
                            logging.info('STDERR: ' + stderr.getvalue())
                if len(errors) > 0:
                    logging.info(f'{test_name} FAILD')
                    num_failed += 1
                else:
                    logging.info(f'{test_name} PASSED')
                    num_passed += 1
    if num_exercises == 0:
        print('{"ok": false, "detail": "No exercises"}')
        return
    if num_submitted == 0:
        print('{"ok": true, "grade": 0, "detail": "No submissions"}')
        return
    if num_passed+num_failed == 0:
        print('{"ok": false, "detail": "No tests run"}')
        return
    points = round(100*(num_submitted/num_exercises)*(num_passed/(num_passed+num_failed)))
    print('{"ok": true, "grade": ' + str(points) + ', "detail": "' +
        str(num_submitted) + '/' + str(num_exercises) + ' submissions, ' +
        str(num_passed)+ '/' + str(num_passed+num_failed) + ' tests passed"}')




if __name__ == '__main__':
    app.run(main)
