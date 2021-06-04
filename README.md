# Programming Education Assistant

[![Build Status](https://travis-ci.org/google/prog-edu-assistant.svg?branch=master)](https://travis-ci.org/google/prog-edu-assistant)

A project to create a set of tools to add autograding capability to
Python programming courses using Jupyter or Colab notebooks.

## Who is this project for?

The main target audience is teaching staff who develop programming or data science courses
using Jupyter notebooks. The tools provided by this project facilitate addition
of autogradable tests to programming assignments, automatic extraction of the
autograding tests and student versions of the notebooks, and easy deployment of
the autograding backend to the cloud.

Initially, the project was started in collaboration with Japanese universities, so
some of the example assignments provided in the `exercises/` subdirectory are in 
the Japanese language. However, anyone using Jupyter notebooks (such as in Colab)
can benefit from this project.

## How to integrate the assistant to your course

There are two main ways to use the assistant:

* Student-driven self-checking with feedback. In this mode, the learner runs the tests interactively
and receives immediate feedback about their code, instead of having to wait for an instructor or 
teaching assistant to give them feedback. Although the tests themselves are not visible to the 
learner, they can run them as many times as desried in order to learn the material, and the results
from test runs are not recorded.

* Grading. By running tests against submitted code and recording the results, the assistant can 
also improve efficiency of instructor workload by automating the task of checking for specific
code syntax and functionality.

Depending your requirements for the above features, you can use either the "pure-Colab" 
approach or the hosted approach.

* Running the autochecking tests inside within the same Python Runtime that student uses. 
Note that this approach only supports self-checking, and cannot be used for _grading_
	student work. See the details in [docs/colab.md](docs/colab.md).

* Hosted on Google Cloud Run. The scripts in this repository provide a server
	and build scripts to build a Docker image that can be deployed to Google
	Cloud Run. The student submissions can be optionally logged.
	See the details in [docs/cloudrun.md](docs/cloudrun.md).

* Manual execution via scripts. This can be used for local grading of student
  submissions against the tests defined in the instructor notebook.
	See the details in [docs/grading.md](docs/grading.md).

The markup format for instructor notebooks is common and described here:

* [exercises/README.md](exercises/README.md).

## Development environment setup

If you want to contribute to this project, start authoring notebooks or
contribute to the project development, see
[SETUP.md](SETUP.md)
for instructions on how to set up the development environment.

## License

Apache-2.0; see [LICENSE](LICENSE) for details.

## Disclaimer

This project is not an official Google project. It is not supported by Google
and Google specifically disclaims all warranties as to its quality,
merchantability, or fitness for a particular purpose.
