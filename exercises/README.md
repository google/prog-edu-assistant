# Programming exercises

This directory contains the programming exercises in the form of the _master
notebooks_. The autograding scripts are automatically extracted from the master
notebooks.

## Installation of the student environment

TODO(salikh): Provide a simpler version of installation instructions for the
student environment sing Conda.

TODO(salikh): Provide a simpler version of student installation instruction
for Colab notebooks.

## Installation of the authoring environment

Install virtualenv. The command may differ depending on the system.

    # On Debian or Ubuntu linux
    apt-get install python-virtualenv  # Install virtualenv.

    # On MacOS with Homebrew:
    brew install python3        # Make sure python3 is installed.
    pip3 install virtualenv     # Install virtualenv.

After that the setup procedure is common

    virtualenv -p python3 ../venv  # Create the virtual Python environment in ./venv/
    source ../venv/bin/activate    # Activate it.
    pip install jupyter            # Install Jupyter (inside of ./venv).

To start the Jupyter notebook run command

    jupyter notebook

There are two more necessary pieces to install:

*   Some tools (utility functions and IPython magics) for using in master
    notebooks, see [../python/prog_edu_assistant_tools/README.md]

*   Jupyter notebook extension for submitting student notebooks, see
    [../nbextensions/upload_it/README.md]

# A template for external integration

This directory contains a skeleton setup for integration
of prog-edu-assistant to an external project.

How to use:

1. Copy the contents of this directory (.ipynb files) into
   an new empty project.

2. Rename `WORKSPACE.ext` to `WORKSPACE`, and rename `BUILD.ext` to `BUILD.bazel` (overwriting the existing file).

2. Run Bazel build to verify that the build setup works.

   ```shell
   bazel build ...
   # If setup works, the following file should be successfully generated.
   ls -l bazel-bin/helloworld-en-student.ipynb
   tar tfi bazel-bin/autograder_tar.tar
   ```

3. Add your assignment notebooks (`.ipynb` files) and add build rules for
   them in `BUILD.bazel`, following the existing example.

4. Replace this `README.md` contents with the description of your project.

## How to build a Docker image

1.  Rename the `docker.ext` directory into `docker`.

    ```shell
    mv docker.ext docker
    ```

2. Run the shell script to build Docker image

    ```shell
    docker/build-docker-image.sh
    ```

3. Verify that the image works locally

    ```shell
    docker/start-local-server.sh
    ```

   Open http://localhost:8000/ to see if the server is ready to accept uploads.

## How to deploy to Google Cloud Run

TODO(salikh): Add local server and Cloud Run deployment instructions here.

## How to author a new assignment

1. Create a new Python 3 notebook in Jupyter.
1. Add a cell with standard imports

    ```python
    # MASTER ONLY

    %load_ext prog_edu_assistant_tools.magics
    from prog_edu_assistant_tools.magics import report, autotest
    ```

1. Pick a new assignment name (it must be unique among the assignment
   names that already exist in the project). Add the following metadata block
   inside a **markdown cell**. Don't omit <code>```</code>.
   
       ```
       # ASSIGNMENT METADATA
       assignment_id: "Variables"
       ```

1. Add some introduction and explanatory material.
1. Add an exercise descrition and metadata in a markdown cell. Use an exercise
   name that is unique inside this assignment notebook.
   
       ```
       # EXERCISE METADATA
       exercise_id: "BigramFrequency"
       ```

1. Add a canonical solution cell right after the exercise metadata cell.

   ```python
   %%solution
   def TopBigramFrequency(k):
       """ # BEGIN PROMPT
       # ... put your program here
       """ # END PROMPT
       # BEGIN SOLUTION
       return {'in the': 100, 'something else': 1}
       # END SOLUTION
   ```

1. Add a student test cell. The student test cell may contain an arbitrary Python code,
typically an assert statement, that students will be able to use for quick self-checking
in their local environment. Note that the `%%studenttest` magic line will not be visible
in the student version of the notebook.

   ```python
   %%studenttest StudentTest
   assert TopBigramFrequency(2)['in the'] == 100
   ```

1. Add an autograder test cell.

   ```python
   %%inlinetest AutograderTest
   assert 'TopBigramFrequency' in globals(), "Did you define a function named 'TopBigramFrequency' in the solution cell?"
   assert str(TopBigramFrequency.__class__) == "<class 'function'>", "Did you define a function named 'TopBigramFrequency'? There was a %s instead" % TopBigramFrequency.__class__
   assert TopBigramFrequency(2)['in the'] == 100
   ```

1. Test the canonical submission with autograder test

   ```python
   result, log = %autotest AutograderTest
   report(AutograderTest, results=result.results)
   ```
   
   TODO(salikh): Add a snippet for asserting that the test passed.

1. Add a submission that is incorrect on purpose.

   ```python
   %%submission
   TopBigramFrequency = 1
   ```

   and test it with the autograder test. The `report()` function imitates the
   HTML output of the autograder that is show to student.

   ```python
   result, log = %autotest AutograderTest
   report(AutograderTest, results=result.results)
   ```

   TODO(salikh): Add a snippet for asserting that the test detected a problem
   and reported it in an expected way.

1. Add a build rule for the assignment notebook to `exercises/BUILD.bazel`:

   ```python
   assignment_notebook(
       name = "nlp-intro",
       src = "nlp-intro-master.ipynb",
   )
   ```

   and add it to the dependency list of autograder_tar target (add
   `":nlp-intro"` to the `deps` list of the rule `autograder_tar`).

   ```python
   autograder_tar(
       name = "autograder_tar",
       deps = [
           # ...
           ":nlp-intro",
       ],
   )
   ```

## Structure of the programming assignment notebooks

Each programming assignment resides in a separate master Jupyter notebook. At
build time, the master notebook is taken as an input and the following outputs
are generated:

*   Student notebook
*   Autograder test directory
*   Automated tests for the notebook
    *   Testing master solution against student tests
    *   Testing master solution against autograder scripts
    *   Testing autograder scripts agains a variety of incomplete and incorrect
        solutions

A student notebook, and by extension, the source master notebook should contain
the following:

*   Explanation of a new concept, algorithm or library
*   Examples of use
*   Explanation of the tasks that the students should complete
*   A solution cell.
    *   In the student notebook the solution is replaced with a prompt of the
        form `... your solution here ...` or similar.
*   A few cells with tests for the student's solution, typically with built-in
    `assert` statements. These are used in two ways:
    *   To test the solution in the master notebook.
    *   To give students a few tests to check their solution.

Each student notebook should have a `assignment_id` entry in the notebook
metadata section that identifies the specific assignment and the course that the
assignment belongs to.

    "metadata": {
      "assignment_id": "Variables",
      # ...
    },

This is useful for deciding which assignment the uploaded notebook is for and
for picking the correct autograder script to run. The metadata is provided in
the master notebook using triple-backtick sections with regexp-friendly markers
in YAML format (which means that the marker itself becomes a YAML comment and is
ignored). `# ASSIGNMENT METADATA` is copied into the notebook-level metadata
field of the student notebook, and `# EXERCISE METADATA` is copied into the cell
level metadata of the next code cell, which designates it as a _solution cell_.

    ```
    # ASSIGNMENT METADATA
    assignment_id: "Variables"
    ```

    ```
    # EXERCISE METADATA
    exercise_id: "DefinePi"
    ```

The solution cell in the master notebook should contain the master solution,
marked with IPython magic `%%solution`. If there is a pair of `# BEGIN SOLUTION`
and `# END SOLUTION` markers, that part will be removed when generating the
student notebook. Otherwise, the whole cell will be replaced by a placeholder.

    %%solution
    PI = 3.14

    %%solution
    def pi():
      # BEGIN SOLUTION
      return 3.14
      # END SOLUTION

The master solution will be replaced with `...` in the student notebook. If a
different replacement is desired, `BEGIN PROMPT` and `END PROMPT` markers may be
used _before_ the SOLUTION block:

    """ # BEGIN PROMPT
    # Define the constant PI here.
    pass
    """ # END PROMPT
    # BEGIN SOLUTION
    PI = 3.14
    # END SOLUTION

The cells that contain student-oriented tests should be marked with `TEST`.
These typically should use Python's `assert` builtin.

    # TEST
    assert(3.1 < PI && PI < 3.2)

The marker `# TEST` is removed when generating the student notebook.

TODO(salikh): Automatically extract `# TEST` cells as unit tests for the master
notebook.

The cells that are autograder scripts should be structured as standard Python
unit tests using the `unittest` module. They need to have markers `BEGIN
UNITTEST` and `END UNITTEST`. Only the lines between the markers are extracted
into autograder scripts. The environment that the unit test expects to find is
provided by the cell magics `%%solution` and `%%submission`. The difference is
that `%%solution` is expected to be correct, so it is executed in the context of
the notebook similar to a regular code cell.

The part of the cell after the `END UNITTEST` marker is also not written to
autograder scripts. It is useful to run the tests in the notebook inline, e.g.
using `autotest` function from the package `prog_edu_assistant_tools`.

## Structure of autograder scripts directories

NOTE: This is a proposed format that is subject to discussion and change.

Autograder tests are the tests that can be run in three environments:

1.  During the assignment authoring in the master Jupyter notebook, to test
    whether the unit tests are catching the expected types of problems
    correctly.
2.  In the automated build system, to check the consistency of the master
    assignment notebooks. This mostly runs the same kind of tests as the master
    notebook, but in an automated manner.
3.  In the autograder worker, against student submissions, to determine the
    grading result.

The autograder scripts have two representations: the directory format and the
notebook format. The notebook format is the authoritative source and is
contained in the master notebook. The directory format is produced at build time
and is included into the autograder image, as well for automated testing of the
notebooks.

### Autograder tests in master notebook

The autograder tests use two special IPython magics. A cell with `%%submission`
cell magic sets up the environment with the given code as hypothetical student
submission. A subsequent cells may use `autotest` function to obtain grading
results from a specific unit test and subsequently check them with assert
statements.

        %%submission
        PI = 4.5

        result, log = autotest(TestPi)
        assert(result.results["TestPi.test_between_3_and_4"] == false)

### Autograder test directories

In the directory format, all autograder scripts take the form of python unit
tests (`*Test.py` files) runnable by the unittest runner. The student's
submission or the master solution will be written into a `submission.py` file
into the scratch directory together with copies of all autograder scripts (unit
tests).

Extraction of the student solution and matching of the solution against unit
tests is done through metadata tags `assignment_id` and `exercise_id`. Following
the linear execution model of Jupyter notebook, all unit tests defined in the
notebook are implicitly assumed to test the last defined exercise.

The exercise directory may also contain a special script `report.py` to to
convert a vector of test outcomes into a human-readable report.

TODO(salikh): Figure out a user-friendly and concise report format.

## List of the exercises

*   `helloworld-en-master.ipynb` --- an example master assignment notebook to
    demonstrate the syntax.

*   `oop-en-master.ipynb` --- master assignment notebook about Object-oriented
    programming.

## Request for contributions

Please add more exercises to this directory!
