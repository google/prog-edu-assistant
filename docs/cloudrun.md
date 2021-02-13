# Running the tests on Google Cloud Run

To enable hosted auto-checking or auto-grading, the auto-checking
tests from the instructor notebook can be extracted into the form
of autograder test directories. Then one needs to build a Docker
image that would include:

* Autograder test directory
* The data necessary for student tests to run
* The upload server [go/cmd/uploadserver]
* Patched NSJail binary

The resulting Docker image can be deployed to Google Cloud Run.
See more information in [docker/README.md]

## Structure of autograder scripts directories

NOTE: This is a proposed format that is subject to discussion and change.

Autograder tests are the tests that can be run in three environments:

1.  During the assignment authoring in the instructor Jupyter notebook, to test
    whether the unit tests are catching the expected types of problems
    correctly.
2.  In the automated build system, to check the consistency of the instructor
    assignment notebooks. This mostly runs the same kind of tests as the instructor
    notebook, but in an automated manner.
3.  In the autograder worker, against student submissions, to determine the
    grading result.

The autograder scripts have two representations: the directory format and the
notebook format. The notebook format is the authoritative source and is
contained in the instructor notebook. The directory format is produced at build time
and is included into the autograder image, as well for automated testing of the
notebooks.

### Autograder tests in instructor notebook

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
submission or the instructor solution will be written into a `submission.py` file
into the scratch directory together with copies of all autograder scripts (unit
tests).

Extraction of the student solution and matching of the solution against unit
tests is done through metadata tags `assignment_id` and `exercise_id`. Following
the linear execution model of Jupyter notebook, all unit tests defined in the
notebook are implicitly assumed to test the last defined exercise.

The exercise directory may also contain a special script `report.py` to to
convert a vector of test outcomes into a human-readable report.

TODO(salikh): Figure out a user-friendly and concise report format.

