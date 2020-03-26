# Design notes.

This file describes the details of technical design of the current implementation.
Everything described here is likely to change as we refine the requirements
and the design.

## Student workflow

This project enables a few different student workflows depending on the chosen setup.

### Student workflow with Colab

Colab is a public free hosted Jupyter-like notebook service by Google that allows
to store Jupyter notebooks in Google Drive and execute them in cloud, requiring
a browser and a Google account on the student's computer. The student's workflow
when using Colab looks like this:

1. A student visits the description page of the course and finds a link to a
	 Colab notebook (Hosting the course description page is out of scope of this
	 project). Student clicks on the link and opens the notebook in Colab in
	 read-only mode. This requires a Google or GSuite account login.

2. The student saves a copy of the notebook to their own Google Drive account.

3. The student clicks on the backend server login link that is embedded in the 
   notebook, uses their Google (GSuite) account to log in to the autograder
	 backend service. The backend services issues a JWT token.

4. The student copies the JWT token, returns back to the Colab notebook and
	 pastes it inside of the notebook.

5. The student works throught the notebook, completes the assignment by typing
   their code into dedicated solution cells.

6. The student submits their solutions for autograding by executing special
   code cells with submission snippet.

7. The server responds with automated feedback. Student modifies the solution
   to address the comments and runs the check again until they are satisfied.

8. Once the student is happy with their results, they submit the notebook
	 for final grading (How to do this is out of scope of this project).

### Student workflow with Jupyter

Jupyter provides a different workflow that runs notebooks locally on student's
computer:

1.  The student installs the client-side environment, including Python 3,
    Jupyter notebook and a custom extension provided by this project. This can
    be done using Virtualenv or Conda.

2.  For each unit of the course, the student downloads the Jupyter notebook with
    course material.

3.  The student reads the description of the assignment within the notebook,
    runs examples and completes the task inside the notebook.

4.  The student presses the "Check" toolbar button in the notebook to upload the
    notebook to the backend system. The backend system stores the notebook in
		temporary storage and runs a few automated tests for the assignment. The
		result of the checks are formatted as an HTML page and shown to the student.
		back to the browser running on student computer and shown in a new tab.

5.  The student modifies the solution to address the comments and runs the check
    again until they are satisfied.

6.  The final submission of the assignment notebook is currently out of scope of
    this project.

## Architecture

This project consists of the following components:

*   Programming assignments in the form of Jupyter notebooks (master
		notebooks).  Notebooks include a few tests, which may be preserved or
		omitted in the student version of the assignment notebook. The key idea is
		that the master notebook is fully self-sufficient, contains all of the
		course materials, student tests, and autograder tests, and successful
		running from top to bottom indicates that all tests pass.

*   The student version of the assignment notebooks is automatically generated
    from the master notebook. Some of the tests are provided for the students to
    check their solution, typically in the form of cells with a few assertions.

*   The tests defined in the master notebooks are also extracted into a
		stand-alone autochecking suite that can be run on server. The results of
		the test run are collected in outcome JSON object and used to render a
		report using a template. The rendered template is returned to student in
		the form of a web page or a HTML snippet.

*   This project provides a Jupyter notebook extension to add a `Check` button
    to the Jupyter notebook that allows to upload the complete assignment
    notebook to the autochecking server.

*   The web server that accepts the notebook uploads, runs automated tests,
    and reports the results back.


## Format of the JSON object returned by the autograder

The top-level object contains the following fields:

* `assignment_id` --- The assignment ID.
* `logs` --- The two-level map of logs outputs, keyed by the exercise ID
  at the first level and by the test name at the second. Logs are intended
    for debugging and not intended to be shown to students.
* `outcomes` --- A map from the test name to the boolean indicating
  whether the test passed or not. False value may indicated failed tests,
    or the test that resulted in error.
* `reports` --- A map from the test name to the HTML string containing the
  rendered student report.
