# Programming Education Assistant

[![Build Status](https://travis-ci.org/google/prog-edu-assistant.svg?branch=master)](https://travis-ci.org/google/prog-edu-assistant)

A project to create a set of tools to add autograding capability to
Python programming courses using Jupyter or Colab notebooks.

## Who is this project for?

The main target audience is teaching staff who develops programming courses
using Jupyter notebooks. The tools provided by this project facilitate addition
of autogradable tests to programming assignments, automatic extraction of the
autograding tests and student versions of the notebooks, and easy deployment of
the autograding backend to the cloud.

The main focus is Japanese universities, so the example assignments provided
in the `exercises/` subdirectory are mostly in Japanese language.

## How to integrate autograder to your course

If you have a course based on Jupyter notebooks and want to integrate the
autochecking tests, there are multiple different way how the autochecking tests
can be run:

* Inside the student notebook (e.g. on Colab). The execution of autochecking
  tests is handled within the same Python Runtime that student uses. Note that
	this approach only supports self-checking, and cannot be used for _grading_
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
