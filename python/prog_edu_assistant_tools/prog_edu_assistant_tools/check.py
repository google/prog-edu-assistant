"""check --- implement the autochecking in Colab easier.

This python package provides a Check() function for autochecking
the Colab notebooks with autograder tests. See
http://github.com/google/prog-edu-assistant for details about
how to add autograder tests to your Colab notebook.
"""

import inspect
import re
import sys
import jinja2

from IPython.core import display
from prog_edu_assistant_tools.magics import report, autotest, CaptureOutput

def GetNotebook():
  """Downloads the ipynb source of Colab notebook"""
  try:
    from google.colab import _message as google_message
  except Exception as e:
    raise Exception('Could not import google_message from google.colab. '
                    'Are you running in Google Colab?\n'
                    'Nested exception: ' + str(e))
  notebook = google_message.blocking_request(
    "get_ipynb", request="", timeout_sec=120)["ipynb"]
  return notebook

def RunInlineTests(submission_source, inlinetests, global_vars=globals()):
  """Runs an inline test."""
  errors = []
  for test_name, test_source in inlinetests.items():
    #print(f'Running inline test {test_name}:\n{test_source}', file=sys.stderr)
    with CaptureOutput() as (stdout, stderr):
      try:
        env = {}
        exec(submission_source, global_vars, env)
        exec(test_source, global_vars, env)
      except AssertionError as e:
        errors.append(str(e))
      if len(stderr.getvalue()) > 0:
        errors.append('STDERR:' + stderr.getvalue())
  if len(errors) > 0:
    results = {'passed': False, 'error': '\n'.join(errors)}
  else:
    results = {'passed': True}
  template_source = """
  <h4 style='color: #387;'>Your submission</h4>
  <pre style='background: #F0F0F0; padding: 3pt; margin: 4pt; border: 1pt solid #DDD; border-radius: 3pt;'>{{ formatted_source }}</pre>
  <h4 style='color: #387;'>Results</h4>
  {% if 'passed' in results and results['passed'] %}
  &#x2705;
  Looks OK.
  {% elif 'error' in results %}
  &#x274c;
  {{results['error'] | e}}
  {% else %}
  &#x274c; Something is wrong.
  {% endif %}"""
  template = jinja2.Template(template_source)
  html = template.render(formatted_source=submission_source, results=results)
  return html

def Check(exercise_id):
  """Checks one exercise against embedded inline tests.

  See documentation in http://github.com/google/prog-edu-assistant
  on the expected format of how the tests are embedded in the ipynb notebook
  metadata.
  """
  def _get_exercise_id(cell):
    if 'metadata' in cell and 'exercise_id' in cell['metadata']:
      return cell['metadata']['exercise_id']
    if 'source' not in cell or 'cell_type' not in cell or cell['cell_type'] != 'code':
      return None
    source = ''.join(cell['source'])
    m = re.search('(?m)^# *EXERCISE_ID: [\'"]?([a-zA-Z0-9_.-]*)[\'"]? *\n', source)
    if m:
      return m.group(1)
    return None
  notebook = GetNotebook()
  # 1. Find the first cell with specified exercise ID.
  found = False
  for (i, cell) in enumerate(notebook['cells']):
    if _get_exercise_id(cell) == exercise_id:
      found = True
      break
  if not found:
    raise Exception(f'exercise {exercise_id} not found')

  submission_source = ''.join(cell['source'])  # extract the submission cell
  submission_source = re.sub(r'^%%(solution|submission)[ \t]*\n', '', submission_source)  # cut %%solution magic
  inlinetests = {}
  if 'metadata' in cell and 'inlinetests' in cell['metadata']:
    inlinetests = cell['metadata']['inlinetests']
  if len(inlinetests) == 0:
    j = i+1
    # 2. If inline tests were not present in metadata, find the inline tests
    # that follow this exercise ID.
    while j < len(notebook['cells']):
      cell = notebook['cells'][j]
      if 'source' not in cell or 'cell_type' not in cell or cell['cell_type'] != 'code':
        j += 1
        continue
      id = _get_exercise_id(cell)
      source = ''.join(cell['source'])
      if id == exercise_id:
        # 3. Pick the last marked cell as submission cell.
        submission_source = source  # extract the submission cell
        submission_source = re.sub(r'^%%(solution|submission)[ \t]*\n', '', submission_source)  # cut %%solution magic
        j += 1
        continue
      m = re.match(r'^%%inlinetest[ \t]*([a-zA-Z0-9_]*)[ \t]*\n', source)
      if m:
        test_name = m.group(1)
        test_source = source[m.end(0):]  # cut %%inlinetest magic
        # 2a. Store the inline test.
        inlinetests[test_name] = test_source
      if id is not None and id != exercise_id:
        # 4. Stop at the next exercise_id.
        break
      j += 1
  # Pick the globals from the caller.
  global_vars = inspect.currentframe().f_back.f_globals
  html = RunInlineTests(submission_source, inlinetests, global_vars)
  return display.HTML(html)
