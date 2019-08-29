# Low-level design notes.

This file describes the low-level technical design of the current prototype.
Everything described here is likely to change as we refine the requirements
and the design.

## Format of the JSON object returned by the autograder:

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
