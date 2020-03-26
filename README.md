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
autograder tests, see
[exercises/README.md](exercises/README.md).

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
