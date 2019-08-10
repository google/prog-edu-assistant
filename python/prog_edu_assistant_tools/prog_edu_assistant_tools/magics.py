"""magics --- make assignment authoring easier.

This python package provides functions and magics that make authoring
of master assignment notebooks more convenient. The functionality it provides:

* Capture canonical solution with %%solution magic
* Anticipate incorrect submissions and test them with %%submission magic
* Create report templates with %%template magic
* Use autotest() and report() functions to run the tests and render
  reports right in the notebook.
"""
import io
import re
import types
import unittest

from IPython.core import display
from IPython.core import magic
import jinja2
import pygments
from pygments import formatters
from pygments import lexers

from prog_edu_assistant_tools import summary_test_result


def autotest(test_class):
    """Runs one unit test and returns the test result.

    This is a non-magic version of the %autotest.
    This function can be used as

        result, log = autotest(MyTestCase)

    Args:
    * test_class: The name of the unit test class (extends unittest.TestCase).

    Returns: A 2-tuple of a SummaryTestResult objct and a string holding verbose
    test logs.
    """
    suite = unittest.TestLoader().loadTestsFromTestCase(test_class)
    errors = io.StringIO()
    result = unittest.TextTestRunner(
        verbosity=4,
        stream=errors,
        resultclass=summary_test_result.SummaryTestResult).run(suite)
    return result, errors.getvalue()


def report(template, **kwargs):
    """Non-magic version of the report.

    This function can be used as

        report(template, source=submission_source.source, results=results)

    The keyword arguments are forwarded to the invocation of
    `template.render()`.  If `source` keyword argument is present, a
    syntax-highlighted HTML copy of it is additionally passed with
    `formatted_source` keyword argument.
    """
    if 'source' in kwargs:
        kwargs['formatted_source'] = pygments.highlight(
            kwargs['source'], lexers.PythonLexer(), formatters.HtmlFormatter())  # pylint: disable=E1101
    # Render the template giving the specified variable as 'results',
    # and render the result as inlined HTML in cell output. 'source' is
    # the prerendered source code.
    return display.HTML(template.render(**kwargs))


# The class MUST call this class decorator at creation time
@magic.magics_class
class MyMagics(magic.Magics):
    """MyMagics -- a collection of IPython magics.

    This class serves as a namespace to hold the IPython magics.
    """
    @magic.line_magic
    def autotest(self, line):
        """Run the unit tests inline

        Returns the TestResult object. Notable fields in the result object:

        * `result.results` is a summary dictionary of
          outcomes, with keys named TestClass.test_case
          and boolean values (True: test passed, False: test failed
          or an error occurred).
          TODO(salikh): Decide whether to remove or document ad-hoc
          entries in the outcome map:
          * 'test_file.py' (set to False if the tests has any issues,
            or set to True if all test cases passed)

        * `result.errors`, `result.failures`, `result.skipped` and other
          fields are computed as documented in unittest.TestResult.
        """

        suite = unittest.TestLoader().loadTestsFromTestCase(
            self.shell.ev(line))
        errors = io.StringIO()
        result = unittest.TextTestRunner(
            verbosity=4,
            stream=errors,
            resultclass=summary_test_result.SummaryTestResult).run(suite)
        return result, errors.getvalue()

    @magic.cell_magic
    def submission(self, line, cell):
        """Registers a submission_source and submission, if the code can run.

        This magic is useful for auto-testing (testing autograder unit tests on
        incorrect inputs)
        """

        _ = line  # Unused.
        # Copy the source into submission_source.source
        self.shell.user_ns['submission_source'] = types.SimpleNamespace(
            source=cell.rstrip())

        env = {}
        try:
            exec(cell, self.shell.user_ns, env)  # pylint: disable=W0122
        except Exception as ex:  # pylint: disable=W0703
            # Ignore execution errors -- just print them.
            print('Exception while executing submission:\n', ex)
            # If the code cannot be executed, leave the submission empty.
            self.shell.user_ns['submission'] = None
            return
        # Copy the modifications into submission object.
        self.shell.user_ns['submission'] = types.SimpleNamespace(**env)

    @magic.cell_magic
    def solution(self, line, cell):
        """Registers solution and evaluates it.

        Also removes the PROMPT block and %%solution from the solution source.

        The difference from %%submission is that the solution is copied into the
        top context,
        making it possible to refer to the functions and variables in subsequent
        notebook cells.
        """
        _ = line  # Unused.

        # Cut out PROMPT
        cell = re.sub(
            '(?ms)^[ \t]*""" # BEGIN PROMPT.*""" # END PROMPT[ \t]*\n?', '',
            cell)
        # Cut out BEGIN/END SOLUTION markers
        cell = re.sub('(?ms)^[ \t]*# (BEGIN|END) SOLUTION[ \t]*\n?', '', cell)

        # Copy the source into submission_source.source
        self.shell.user_ns['submission_source'] = types.SimpleNamespace(
            source=cell.rstrip())

        # Capture the changes produced by solution code in env.
        env = {}
        # Note: if solution throws exception, this breaks the notebook
        # execution, and this is intended. Solution must be correct!
        exec(cell, self.shell.user_ns, env)  # pylint: disable=W0122
        # Copy the modifications into submission object.
        self.shell.user_ns['submission'] = types.SimpleNamespace(**env)
        # Copy the modifications into user_ns
        for k in env:
            self.shell.user_ns[k] = env[k]

    @magic.cell_magic
    def template(self, line, cell):
        """Registers a template for report generation.

        Args:
        * line: The string with the contents of the remainder of the line
          starting with %%template.
        * cell: The string with the contents of the code cell, excluding
          the line with %%template marker.

        Hint: Use {{results['TestClassName.test_method']}} to extract specific
        outcomes and be prepared for the individual test case keys to be absent
        in the results map, if the test could not be run at all (e.g. because of
        syntax error in the submission). The corresponding test outcome map
        should be passed to the template invocation with keyword argument:
        `results=result.results` where `result` is an instance of
        `SummaryTestResult` returned by `autotest()`.

        Warning: %%template must not use triple-quotes inside.
        """
        name = line
        if line == '':
            name = 'report_template'
        if re.search('"""', cell):
            raise Exception("%%template must not use triple-quotes")
        # Define a Jinja2 template based on cell contents.
        self.shell.user_ns[name] = jinja2.Template(cell)

    @magic.cell_magic
    def report(self, line, cell):
        """Renders the named template.

        Syntax:
          %%report results_var
          template_name
        """
        var_name = line
        template_name = cell
        template = self.shell.ev(template_name)
        results = self.shell.ev(var_name)
        source = self.shell.user_ns['submission_source'].source
        formatted_source = pygments.highlight(
            source,
            lexers.PythonLexer(),  # pylint: disable=E1101
            formatters.HtmlFormatter())  # pylint: disable=E1101
        # Render the template giving the specified variable as 'results',
        # and render the result as inlined HTML in cell output. 'source' is
        # the prerendered source code.
        return display.HTML(
            template.render(results=results,
                            source=source,
                            formatted_source=formatted_source))


def load_ipython_extension(ipython):
    """This function is called when the extension is

    loaded. It accepts an IPython InteractiveShell
    instance. We can register the magic with the
    `register_magic_function` method of the shell
    instance.
    """
    ipython.register_magics(MyMagics)
