# Programming Education Assistant

[![Build Status](https://travis-ci.org/google/prog-edu-assistant.svg?branch=master)](https://travis-ci.org/google/prog-edu-assistant)

A project to create a suite of programming assignments that can be used for
running a college-level programming course. The tools provided in this
repository make it easier to author new assignments as Jupyter notebooks,
automatically generate the student version of the assingment notebooks, and
provide the server-side auto-checking for student notebooks.

The main focus is Japanese universities, so the assignments text may be
developed or translated to Japanese language.

## Development environment

If you want to start authoring notebooks or contribute to the project
development, see [SETUP.md] for setup instructions.

## Student workflow

The followint student workflow is intended for the programming assignments
included with this project:

1.  The student installs the client-side environment, including Python 3,
    Jupyter notebook and a custom extension provided by this project. This can
    be done using Virtualenv or Conda.

2.  For each unit of the course, the student downloads the Jupyter notebook with
    course material about 1 week in advance of the class.

3.  The student reads the description of the assignment within the notebook,
    runs examples and completes the task inside the notebook.

4.  The student presses the "Check" menu item in the notebook to upload the
    notebook to the backend system. The backend system stores the notebook in
    temporary storage and runs a few automated checks, including syntax and
    style checker and runtime tests for the assignment. The result of the checks
    are formatted as an HTML page and the URL is passed back to the browser
    running on student computer and shown in a new tab.

5.  The student modifies the solution to address the comments and runs the check
    again until they are satisfied.

6.  The final submission of the assignment notebook is currently out of scope of
    this project.

## Architecture

This project consists of the following components:

*   Programming assignments in the form of Jupyter notebooks (master notebooks).
    Notebooks include a few tests, which may be preserved or omitted in the
    student version of the assignment notebook. The key idea is that the master
    notebook is fully self-sufficient, and successful running from top to bottom
    indicates that all tests pass.

*   The student version of the assignment notebooks are automatically generated
    from the master notebook. Some of the tests are provided for the students to
    check their solution, typically in the form of cells with a few assertions.

*   The tests defined in the master notebooks are also extracted into a
    stand-alone autochecking suite that can be run on server. The results of the
    test run are collected in outcome map and used to render a report from a
    template. The rendered template is returned to student in the form of a web
    page.

*   This project provides a Jupyter notebook extension to add a `Check` button
    to the Jupyter notebook that allows to upload the complete assignment
    notebook to the autochecking server.

*   The web server that accepts the notebook uploads and schedules checker runs,
    and reports the results back.

*   The server backend that runs scheduled checkers (worker). The worker and the
    web server use a message queue to communicate.

## License

Apache-2.0; see [LICENSE](LICENSE) for details.

## Disclaimer

This project is not an official Google project. It is not supported by Google
and Google specifically disclaims all warranties as to its quality,
merchantability, or fitness for a particular purpose.
