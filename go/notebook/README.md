# Assignment builder

This directory contains code for the assignment builder, which is a tool that
takes a master notebook as input, and produces a student notebook as well as a
autograder directory with the tests.

This tool is very similar to jassign (https://github.com/okpy/jassign), but the
requirement set is slightly different, and jassign seems to be a quite young
project, so it did not look appropriate to reuse. However, given the goals are
very similar, it is possible we may consider reusing jassign or merging to it.
Parts of the syntax has been made compatible with jassign.

## Requirements to the assignment builder

*   A single master notebook is the source of all deliverables:
    *   Student notebook (to be distributed to students)
    *   Notebook tests (to be run at the build time to check notebook
        correctness)
    *   Autograder scripts (to be copied into the autograder image to be used
        for autograding student submissions).
*   Running the notebook from top to bottom should be possible and should run
    the actual tests and check the correctness, so if any of the tests fail, the
    notebook cell should throw an exception. A successful completion of the
    whole notebook should indicate absence of any problem that the tests can
    detect.
*   Running the notebook tests should be equivalent to running the notebook.
*   The master notebook should contain some tests to check correctness of
    autograding scripts.

## Syntax details

### MASTER ONLY

Any cell that matches `# MASTER ONLY` will be removed when generating a student
notebooks.

### Solution

Solution cells must be marked with `%%solution` cell magics.

    %%solution
    x = 2

    %%solution
    def f():
      # BEGIN SOLUTION
      return 2
      # END SOLUTION

TODO(salikh): Implement a special marker (`# UNITTEST OUTPUT`?) that instructs
that the stdout output of the master solution should be recorded and used for
testing student submissions.

The solutions are removed when producing a student notebook, with the
replacement being either a generic "...", or provided with `PROMPT` markers.

    """  # BEGIN PROMPT
    # Put your code here (and remove 'pass').
    pass
    """  # END PROMPT

### Solution tests

A few tests can be provided in a notebook to quickly check if the solution
satisfies basic correctness criteria. This tests are marked with `# TEST` and
are preserved in the student notebook. They are also extracted into standalone
notebook tests.

    # TEST

### Unit tests / autograder scripts

These tests are intended to be used for autograding, i.e. running the tests on
potentially incorrect solution in order to identify problem points and give some
feedback to students. They are marked with `UNITTEST` and are used in two ways:

*   To test the master solution (all unit tests should pass on the master
    solution).
*   To extract the autograder scripts.

Note: to test the autograder scripts, the second-meta-level tests provide
various incorrect inputs, and the unit tests are run and the outcome vector is
checked against the expected one.

### Autograder tests (self-tests)

Autograder tests are performed by providing an potentially incorrect submission
using `%%submission` cell magic, and running the unit test using `autotest`
functionality. The resulting outcome vector can be checked using plain Python
`assert` statements.

### Report scripts

Report scripts are used by the autograder to provide human-readable feedback
without necessarily revealing the autograder tests themselves.
