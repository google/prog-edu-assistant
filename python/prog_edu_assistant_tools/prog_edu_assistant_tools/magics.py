import io
import re
import sys
import unittest

from IPython.core.display import HTML
from IPython.core.magic import (Magics, magics_class, line_magic, cell_magic,
                                line_cell_magic)
from jinja2 import Template
from prog_edu_assistant_tools.summary_test_result import SummaryTestResult
from pygments import formatters
from pygments import highlight
from pygments.lexers import PythonLexer
from types import SimpleNamespace


def autotest(testClass):
    """Runs one unit test and returns the test result.

    This is a non-magic version of the %autotest.
    This function can be used as

        result, log = autotest(MyTestCase)

    Returns: A 2-tuple of a SummaryTestResult objct and a string holding verbose
    test logs.
    """
    suite = unittest.TestLoader().loadTestsFromTestCase(testClass)
    errors = io.StringIO()
    result = unittest.TextTestRunner(verbosity=4,
                                     stream=errors,
                                     resultclass=SummaryTestResult).run(suite)
    return result, errors.getvalue()


def report(template, **kwargs):
    """Non-magic version of the report.

    This function can be used as

        report(template, source=submission_source.source, results=results)

    The keyword arguments are forwarded to the invocation of `template.render()`.
    The `source` keyword argument is piped through syntax highlighter before
    forwarding, and the original raw source is instead passed as `raw_source`
    keyword argument.
    """
    if 'source' in kwargs:
        # TODO(salikh): Avoid rewriting user input.
        kwargs['raw_source'] = kwargs['source']
        kwargs['source'] = highlight(kwargs['source'], PythonLexer(),
                                     formatters.HtmlFormatter())
    # Render the template giving the specified variable as 'results',
    # and render the result as inlined HTML in cell output. 'source' is
    # the prerendered source code.
    return HTML(template.render(**kwargs))


# The class MUST call this class decorator at creation time
@magics_class
class MyMagics(Magics):
    @line_magic
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
            verbosity=4, stream=errors,
            resultclass=SummaryTestResult).run(suite)
        return result, errors.getvalue()

    def lmagic(self, line):
        'my line magic'
        print('Full access to the main IPython object:', self.shell)
        print('Variables in the user namespace:',
              list(self.shell.user_ns.keys()))
        return line

    def cmagic(self, line, cell):
        'my cell magic'
        return line, cell

    @cell_magic
    def submission(self, line, cell):
        """Registers a submission_source and submission, if the code can run.

        This magic is useful for auto-testing (testing autograder unit tests on
        incorrect inputs)
        """

        # Copy the source into submission_source.source
        self.shell.user_ns['submission_source'] = SimpleNamespace(
            source=cell.rstrip())

        env = {}
        try:
            exec(cell, self.shell.user_ns, env)
        except Exception as e:
            # Ignore execution errors -- just print them.
            print('Exception while executing submission:\n', e)
            # If the code cannot be executed, leave the submission empty.
            self.shell.user_ns['submission'] = None
            return None
        # Copy the modifications into submission object.
        self.shell.user_ns['submission'] = SimpleNamespace(**env)

    @cell_magic
    def solution(self, line, cell):
        """Registers solution and evaluates it.

        Also removes the PROMPT block and %%solution from the solution source.

        The difference from %%submission is that the solution is copied into the
        top context,
        making it possible to refer to the functions and variables in subsequent
        notebook cells.
        """

        # Cut out PROMPT
        cell = re.sub(
            '(?ms)^[ \t]*""" # BEGIN PROMPT.*""" # END PROMPT[ \t]*\n?', '',
            cell)
        # Cut out BEGIN/END SOLUTION markers
        cell = re.sub('(?ms)^[ \t]*# (BEGIN|END) SOLUTION[ \t]*\n?', '', cell)

        # Copy the source into submission_source.source
        self.shell.user_ns['submission_source'] = SimpleNamespace(
            source=cell.rstrip())

        env = {}
        # Note: if solution throws exception, this breaks the execution. Solution must be correct!
        # TODO(salikh): Use self.shell.ex() instead of exec().
        exec(cell, self.shell.user_ns, env)
        # Copy the modifications into submission object.
        self.shell.user_ns['submission'] = SimpleNamespace(**env)
        # Copy the modifications into user_ns
        for k in env:
            self.shell.user_ns[k] = env[k]

    @cell_magic
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
        """
        name = line
        if line == '':
            name = 'report_template'
        # Define a Jinja2 template based on cell contents.
        self.shell.user_ns[name] = Template(cell)

    @cell_magic
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
        highlighted_source = highlight(
            self.shell.user_ns['submission_source'].source, PythonLexer(),
            formatters.HtmlFormatter())
        # Render the template giving the specified variable as 'results',
        # and render the result as inlined HTML in cell output. 'source' is
        # the prerendered source code.
        return HTML(template.render(results=results,
                                    source=highlighted_source))


def load_ipython_extension(ipython):
    """This function is called when the extension is

    loaded. It accepts an IPython InteractiveShell
    instance. We can register the magic with the
    `register_magic_function` method of the shell
    instance.
    """
    ipython.register_magics(MyMagics)
