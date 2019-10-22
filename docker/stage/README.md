# Worker

This directory contains a Docker file and source files necessary
to build a container image of autograder worker.

Note: the autograder scripts are not present in this directory,
because they are necessarily assignment-dependent and so are
authored together with the assignment itself in the form of
a Jupyter notebook with special markup.

The autograder scripts are built and shipped into the container
image of the autograder worker.

## Architecture

To simplify the initial demo, the autograder worker is decoupled
from the main system as much as possible. We chose to use RabbitMQ
as the communication mechanism. The autograder worker is configured
with just one string that is the spec of the RabbitMQ instance.

Workers read from the "submissions" queue, and assume that each
submission is a full Jupyter notebook submission, i.e. tries to
parse all received messages as JSON. If JSON parsing fails, the worker
will drop the submission.

For the decoded submissions, the worker will detect which assignment
it corresponds to, extract the submitted code from the notebook
cell, prepare test directories and run the test suites in each.
It will then analyze the output of test suite and build the outcome
vector, and then generate the test report based on the input
submission and the outcome vector.

To minimize coupling of worker with the rest of the system, the worker
uses the same RabbitMQ to post the report on to the queue "reports".
The report is posted in JSON format, with snippets of HTML inside.
