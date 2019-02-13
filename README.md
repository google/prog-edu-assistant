# Programming Education Assistant

A project to create a suite of programming assignments
that can be used for running a college-level programming
course.

The main focus is Japanese universities, so the assignments
text may be developed or translated to Japanese language.

The initial release of this project is empty and includes
only boilerplate (license, contributor guidelines and this
README file).

## Intended student workflow

The followint student workflow is intended for the programming
assignments included with this project:

0 The student installs the client-side environment, including
  Python 3, Jupyter notebook and a custom extension provided
  by this project.

1 For each unit of the course, the student downloads the
  Jupyter notebook with course material about 1 week in
  advance of the class.

2 The student reads the description of the assignment within
  the notebook, runs examples and completes the task inside
  the notebook.

3 The student presses the "Check" menu item in the notebook
  to upload the notebook to the backend system. The backend
  system stores the notebook in temporary storage and runs
  a few automated checks, including syntax and style checker
  and runtime tests for the assignment. The result of
  the checks are formatted as code comments and passed back
  to the Jupyter notebook environment on the student's
  computer.

4 The notebook shows the message received from the checker
  to the student, perhaps by adding a temporary cell with
  rendered code and checker findings.

5 The student modifies the solution to address the comments
  and runs the check again until they are satisfied.

6 The student finally selects the "Submit" menu item in
  the notebook and makes a formal submission.

## Architecture

This project is planned to have the following components:

* Student assignments in the form of self-study Jupyter
  notebooks. Notebooks may include a few tests that allow
  the student to check whether their solution produces correct
  answers for a few test inputs. Each assignment is
  a stand-alone Jupyter notebook that can be run without
  any other components of the system.

* For each assignment notebook there exists a checker suite
  that consists of (1) solution extractor, (2) syntax checker,
  (3) style checker, (4) a set of automated test cases,
  (5) an annotator that takes the output of the rest of the
  components, selects the most important one and formats it
  as a finding associated with a code location. The 
  output format for the checkers may be similar to the
  compiler or lint error output, i.e. it may locate
  the line and column of where the message applies.
  The message may also be a non-located, e.g. if it is
  based on a specific test failure pattern, perhaps indicating
  that some edge case has not been handled.

* Jupyter notebook extension to enable `Check` and `Submit`
  buttons, as well as presenting the messages from the checker
  back to the student.

* The web server that accepts the notebook uploads and
  schedules checker runs, and reports the results back.

* The server backend that runs scheduled checkers.

## License

Apache-2.0; see [LICENSE](LICENSE) for details.

## Disclaimer

This project is not an official Google project. It is not
supported by Google and Google specifically disclaims all
warranties as to its quality, merchantability, or fitness for
a particular purpose.
