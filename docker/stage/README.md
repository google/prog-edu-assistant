# Dockerfile.combined

This is a Docker file to build a combined image with both
HTTP server and the autograder, so that grading happen
synchronously to fit Google Cloud Run execution model.

Note: the autograder scripts are not present in this directory,
because they are necessarily assignment-dependent and so are
authored together with the assignment itself in the form of
a Jupyter notebook with special markup.

The autograder scripts are built and shipped into the container
image of the autograder worker.

## Architecture

The web server takes upload requests at the endpoint:

    /upload

The specific port is provided by Cloud Run runtime via `PORT` environment
variable. The endpoint accepts form file uploads, and tries to parse them as
JSON .ipynb files. If JSON parse fails, the server drops the submission. If
JSON parse succeeds, it decides which assignment the upload is related to by
looking at notebook metadata (looking for `assignment_id`).

Then the server extracts the submitted code from the notebook
assignment cells, prepares test directories and runs the test suites in each.
It then analyzes the output of test suite and builds the outcome
JSON object, and then generate the test report.

The server works in a synchronous manner to enable autoscaling
by Cloud Run, so the requests typically take a few seconds to complete.
