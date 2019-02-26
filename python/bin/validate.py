#!/usr/bin/env python3
'''validate.py run unit tests, lint and code format checks.'''

import glob
import os
import subprocess
import sys

from absl import app
from absl import flags

FLAGS = flags.FLAGS

flags.DEFINE_bool('update', False, '')


def _run_cmd(args):
    print('\x1b[1;34m$$ ' + ' '.join(args) + '\x1b[0m')
    return subprocess.run(args).returncode


def _print_error(msg):
    print('\x1b[1;31m' + msg + '\x1b[0m', file=sys.stderr)


def _print_ok(msg):
    print('\x1b[1;32m' + msg + '\x1b[0m', file=sys.stderr)


def main(_):
    '''the entry point'''
    root = os.path.abspath(os.path.join(os.path.dirname(__file__), '..'))
    os.chdir(root)
    py_files = glob.glob('**/*.py', recursive=True)

    failed = False

    rcode = _run_cmd(['pylint'] + py_files)
    if rcode:
        _print_error('pylint detected errors.')
        failed = True
    else:
        _print_ok('passed pylint check')

    yapf_cmd = ['yapf'] + py_files
    if FLAGS.update:
        yapf_cmd.append('--in-place')
    else:
        yapf_cmd.append('--diff')
    rcode = _run_cmd(yapf_cmd)
    if rcode:
        _print_error('yapf detected illformed code. Run yapf with --in-place '
                     'to format code')
        failed = True
    else:
        _print_ok('passed yapf check')

    rcode = _run_cmd(['python3', '-m', 'utils_test'])
    if rcode:
        _print_error('unit tests failed')
        failed = True
    else:
        _print_ok('all unit tests passed')

    if failed:
        return 1
    return 0


if __name__ == '__main__':
    app.run(main)
