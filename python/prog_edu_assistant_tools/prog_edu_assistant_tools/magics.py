"""magics --- make assignment authoring easier.

This python package provides functions and magics that make authoring
of master assignment notebooks more convenient. The functionality it provides:

* Capture canonical solution with %%solution magic
* Anticipate incorrect submissions and test them with %%submission magic
* Create report templates with %%template magic
* Use autotest() and report() functions to run the tests and render
  reports right in the notebook.
"""
import ast
import io
import re
import types
import unittest
import sys
import io

from contextlib import contextmanager
from io import StringIO

from IPython.core import display
from IPython.core import magic

import jinja2
import pygments

from pygments import formatters
from pygments import lexers

from prog_edu_assistant_tools import summary_test_result


@contextmanager
def CaptureOutput():
    """Captures the stdout and stderr into StringIO objects."""
    capture_out, capture_err = StringIO(), StringIO()
    save_out, save_err = sys.stdout, sys.stderr
    try:
        sys.stdout, sys.stderr = capture_out, capture_err
        yield sys.stdout, sys.stderr
    finally:
        sys.stdout, sys.stderr = save_out, save_err

def autotest(test_case):
    """Runs one unit test and returns the test result.

    This is a non-magic version of the %autotest.
    This function can be used as

        result, log = autotest(MyTestCase)

    Args:
    * test_case: The name of the unit test class (extends unittest.TestCase),
                 or the name of the inline test namespace, defined with
                 %%inlinetest or %%studenttest.

    Returns: A 2-tuple of a SummaryTestResult objct and a string holding verbose
    test logs.
    """
    if isinstance(test_case, type) and issubclass(test_case, unittest.TestCase):
        suite = unittest.TestLoader().loadTestsFromTestCase(test_case)
        errors = io.StringIO()
        result = unittest.TextTestRunner(
            verbosity=4,
            stream=errors,
            resultclass=summary_test_result.SummaryTestResult).run(suite)
        return result, errors.getvalue()
    elif (isinstance(test_case, types.SimpleNamespace) and
            test_case.type == 'inlinetest'):
        errorMessage = None
        with CaptureOutput() as (out, err):
            try:
                env = {}
                # Assume the context has already been
                # set up in user_ns.
                # Exercise the code under test.
                exec(globals()['submission_source'].source,
                        globals(), env)  # pylint: disable=W0122
                # Exercise the inline test.
                exec(test_case.source, globals(), env)  # pylint: disable=W0122
            except AssertionError as e:
                errorMessage = e
            except Exception as e:
                errorMessage = e
        stream = out.getvalue()
        if len(err.getvalue()) > 0:
            stream += "\nERROR:\n" + err.getvalue()
        result = summary_test_result.SummaryTestResult(
                stream=stream,
                descriptions=test_case.name,
                verbosity=1)
        if errorMessage is not None:
            result.results['error'] = errorMessage
            result.results['passed'] = False
        else:
            result.results['passed'] = True
        return result, stream
    else:
        raise Exception("Unrecognized autotest argument of class %s" % (test_case.__class__))



def report(template, **kwargs):
    """Non-magic version of the report.

    This function can be used as

        report(template, source=submission_source.source, results=results)

    The keyword arguments are forwarded to the invocation of
    `template.render()`.  If `source` keyword argument is present, a
    syntax-highlighted HTML copy of it is additionally passed with
    `formatted_source` keyword argument.

    Args:
      template - A template earlier defined with %%template magic,
                 or an inline test. In case of an inline test, the template
                 is automatically defined to include the source code (if provided)
                 and the error message from the inline test result.

    Returns:
      A displayable HTML object.
    """
    if 'source' in kwargs:
        kwargs['formatted_source'] = pygments.highlight(
                kwargs['source'],
                lexers.PythonLexer(),  # pylint: disable=E1101
                formatters.HtmlFormatter()).rstrip()  # pylint: disable=E1101
    # Render the template giving the specified variable as 'results',
    # and render the result as inlined HTML in cell output. 'source' is
    # the prerendered source code.
    if isinstance(template, jinja2.Template):
        html = template.render(**kwargs)
    elif isinstance(template, types.SimpleNamespace) and template.type == 'inlinetest':
        source_template = """
<h4 style='color: #387;'>Your submission</h4>
<pre style='background: #F0F0F0; padding: 3pt; margin: 4pt; border: 1pt solid #DDD; border-radius: 3pt;'>{{ formatted_source }}</pre>"""
        result_template = """
<h4 style='color: #387;'>Results</h4>
{% if 'passed' in results and results['passed'] %}
Looks OK.
{% elif 'error' in results %}
{{results['error'] | e}}
{% else %}
Something is wrong.
{% endif %}"""
        if 'formatted_source' in kwargs:
            template_source = source_template
        else:
            template_source = ''
        template_source += result_template
        actual_template = jinja2.Template(template_source)
        html = actual_template.render(**kwargs)
    else:
        raise Exception("Unrecognized template argument of class %s" %
                (test_case.__class__))
    return display.HTML(html)


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

        test_case = self.shell.ev(line)
        if isinstance(test_case, type) and issubclass(test_case, unittest.TestCase):
            suite = unittest.TestLoader().loadTestsFromTestCase(
                test_case)
            errors = io.StringIO()
            result = unittest.TextTestRunner(
                verbosity=4, stream=errors,
                resultclass=summary_test_result.SummaryTestResult).run(suite)
            return result, errors.getvalue()
        elif (isinstance(test_case, types.SimpleNamespace) and
                test_case.type == 'inlinetest'):
            errorMessage = None
            with CaptureOutput() as (out, err):
                try:
                    # Assume the context has already been
                    # set up in user_ns.
                    env = {k: v for (k, v) in self.shell.user_ns.items()}
                    # Exercise the code under test.
                    exec(self.shell.user_ns['submission_source'].source,
                            env)  # pylint: disable=W0122
                    # Exercise the inline test.
                    exec(test_case.source, env)  # pylint: disable=W0122
                except AssertionError as e:
                    errorMessage = e
                except Exception as e:
                    errorMessage = e
            stream = out.getvalue()
            if len(err.getvalue()) > 0:
                stream += "\nSTDERR:\n" + err.getvalue()
            result = summary_test_result.SummaryTestResult(
                    stream=stream,
                    descriptions=test_case.name,
                    verbosity=1)
            if errorMessage is not None:
                result.results['error'] = errorMessage
                result.results['passed'] = False
            else:
                result.results['passed'] = True
            return result, stream
        else:
            raise Exception("Unrecognized autotest argument of class %s" % (test_case.__class__))

    @classmethod
    def CutPrompt(cls, cell):
        '''Returns the contents of the cell cleaned from markup.

        It cuts out the prompt blocks (""" # BEGIN/END PROMPT), and drops the
        solution marker lines (# BEGIN/END SOLUTION).
        '''
        mm = [m for m in re.finditer(
            '(?ms)^[ \t]*""" # (BEGIN) PROMPT[ \t]*\n?|""" # (END) PROMPT[ \t]*\n?',
            cell)]
        last = -1  # The end of the block to cut, or -1 if not in prompt region.
        # Process the list of matches backwards so that cutting positions
        # would not be affected.
        mm.reverse()
        for m in mm:
          if m.groups()[1] == "END":
            if last != -1:
              raise Exception(f'Unbalanced PROMPT block in cell:\n{cell}')
            last = m.end()
          else:
            if last == -1:
              raise Exception(f'Unbalanced PROMPT block in cell:\n{cell}')
            cell = cell[0:m.start()] + cell[last:]
            last = -1
        if last != -1:
          raise Exception(f'Unbalanced PROMPT block in cell:\n{cell}')
        # Cut out BEGIN/END SOLUTION markers
        cell = re.sub('(?ms)^[ \t]*# BEGIN SOLUTION[ \t]*\n?', '', cell)
        cell = re.sub('(?ms)^[ \t]*# END SOLUTION[ \t]*\n?', '', cell)
        return cell


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

        # Cut out PROMPT and SOLUTION markers.
        cell = self.CutPrompt(cell)

        # Copy the source into submission_source.source
        self.shell.user_ns['submission_source'] = types.SimpleNamespace(
            source=cell.rstrip())

        # Capture the changes produced by solution code in env.
        env = {}
        # Note: if solution throws exception, this breaks the notebook
        # execution, and this is intended. Solution must be correct!
        # Hack: Peel the last expression to eval.
        ret = None
        block = ast.parse(cell, mode='exec')
        # Check if the last statement of the block is an assignment.
        if block.body[-1].__class__ != ast.Assign and block.body[-1].__class__ != ast.FunctionDef:
            # If not an assignment, eval last expression as opposed to exec.
            try:
                last_expr = ast.Expression(block.body.pop().value)
                exec(compile(block, '', mode='exec'), self.shell.user_ns, env)  # pylint: disable=W0122
                ret = eval(compile(last_expr, '', mode='eval'),
                           self.shell.user_ns, env)
            except Exception as e:
                print(e)
                # Give up on recovering the last value, just execute.
                exec(cell, self.shell.user_ns, env)  # pylint: disable=W0122
        else:
            # Otherwise just exec the whole block and do not bother
            # about the return value.
            exec(cell, self.shell.user_ns, env)  # pylint: disable=W0122
        # Copy the modifications into submission object.
        self.shell.user_ns['submission'] = types.SimpleNamespace(**env)
        # Copy the modifications into user_ns
        for k in env:
            self.shell.user_ns[k] = env[k]
        return ret

    @magic.cell_magic
    def inlinetest(self, line, cell):
        """Registers an inline test.

        An inline test is a piece of code that can be executed either directly
        in the Jupyter notebook, or extracted into an automated test that 
        takes the notebook context into account.
        """

        name = line
        if not re.fullmatch(r'[a-zA-Z][a-zA-Z0-9_]*', name):
            raise Exception("%%inlinetest must use an identifier as a name, "
                            "got %s" % name)

        # Copy the source into the variable 
        self.shell.user_ns[name] = types.SimpleNamespace(
            source=cell.rstrip(), type='inlinetest', name=name)

        # Capture the changes produced by inline test code in env.
        env = {}
        # Note: if inline test throws exception, this breaks the notebook
        # execution, and this is intended. Inline tests emulate direct
        # execution of the code.
        exec(cell, self.shell.user_ns, env)  # pylint: disable=W0122
        # Copy the modifications into user_ns
        for k in env:
            self.shell.user_ns[k] = env[k]

    @magic.cell_magic
    def studenttest(self, line, cell):
        """Registers an inline test.

        An inline test is a piece of code that can be executed either directly
        in the Jupyter notebook, or extracted into an automated test that 
        takes the notebook context into account.
        """

        name = line
        if not re.fullmatch(r'[a-zA-Z][a-zA-Z0-9_]*', name):
            raise Exception("%%studenttest must use an identifier as a name, "
                            "got %s" % name)

        # Copy the source into the variable. Note: %%studenttest
        # is the same as %%inlinetest, with the only difference being that
        # it is preserved in the student notebook.
        self.shell.user_ns[name] = types.SimpleNamespace(
            source=cell.rstrip(), type='inlinetest', name=name)

        # Capture the changes produced by inline test code in env.
        env = {}
        # Note: if inline test throws exception, this breaks the notebook
        # execution, and this is intended. Inline tests emulate direct
        # execution of the code.
        exec(cell, self.shell.user_ns, env)  # pylint: disable=W0122
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
        return display.HTML(template.render(
            results=results,
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
