#@title Submission snippet
SUBMISSION_URL = 'https://combined-v6gvzmyosa-an.a.run.app/upload' #@param

# NOTE: The authoritative original source of this snipped is in
# github.com/google/prog-edu-assistant/student/preamble.py.

import json
import requests

from google.colab import _message as google_message
from IPython.core import display

def Submit(submission_url=SUBMISSION_URL):
  notebook = google_message.blocking_request(
    'get_ipynb', request='', timeout_sec=120)['ipynb']
  data = json.dumps(notebook)
  r = requests.post(submission_url, files={'notebook': data})
  # TODO(salikh): Implement
  display.display(display.HTML(r.content.decode('utf-8')))
