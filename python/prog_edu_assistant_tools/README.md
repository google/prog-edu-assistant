# Notebook tools

This package contains a few functions that makes authoring programming assignments in Jupyter
more convenient, including the following:

* A summarizing test runner to run unit tests
* A function that can run a unit test directly in the Jupyter notebook

In short, the intention of this package is to hide all the smarts
of having a complete master notebook capable of complete auto-testing
behind nice and readable names that can be imported rather than reimplemented
in every master notebook.

## How to build this package

    source ../../../venv/bin/activate
    python setup.py bdist_wheel

## How to install this package locally

    pip install dist/prog_edu_assistant_tools-0.1-py3-none-any.whl

If you are a developer and want to reinstall the package, use the following:

    pip install --ignore-installed dist/prog_edu_assistant_tools-0.1-py3-none-any.whl

## How to use this package in master notebooks.

Here are some useful snippets for your master assignment notebooks:

    from prog_edu_assistant_tools.summary_test_result import SummaryTestResult
    from prog_edu_assistant_tools.magics import autotest, report

    # Loads %%solution, %%submission, %%template
    %load_ext prog_edu_assistant_tools.magics

