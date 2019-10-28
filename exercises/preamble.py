# NOTE: The authoritative original source of this snipped is in
# github.com/google/prog-edu-assistant/student/preamble.py.

#@title Submission snippet
#@markdown Please [login to server](https://combined-v6gvzmyosa-an.a.run.app/login) to get a JWT token and paste it here.
SERVER_URL = 'https://combined-v6gvzmyosa-an.a.run.app'
JWT_TOKEN = "" #@param

import json
import requests

from google.colab import _message as google_message
from IPython.core import display

def Submit(exercise_id=None):
  if JWT_TOKEN == "":
    display.display(display.HTML("Please get JWT_TOKEN by visiting " +
                                 "<a href='" + SERVER_URL + "/login'>Login page</a>"))
    raise Exception("Please set JWT_TOKEN")
  notebook = google_message.blocking_request(
    "get_ipynb", request="", timeout_sec=120)["ipynb"]
  ids = []
  for cell in notebook['cells']:
        if 'metadata' not in cell:
          continue
        m = cell['metadata']
        if m and 'exercise_id' in m:
            cell_id = m['exercise_id']
            if cell_id:
                ids.append(cell_id)
  params = {}
  if exercise_id:
    if exercise_id not in ids:
        raise Exception('Not valid exercise ID: ' + exercise_id + ". Valid ids: " + ", ".join(ids))
    params["exercise_id"] = exercise_id
  data = json.dumps(notebook)
  r = requests.post(SERVER_URL + "/upload", files={"notebook": data},
                    headers={"Authorization": "Bearer " + JWT_TOKEN},
                    params=params)
  if r.status_code == 401:
    display.display(display.HTML("Not authorized: is your JWT_TOKEN correct? " +
                                 "Please get JWT_TOKEN by visiting " +
                                 "<a target='_blank' href='" + SERVER_URL + "/login'>Login page</a>" +
                                 "in a new browser tab."))  
  display.display(display.HTML(r.content.decode('utf-8')))
  
if JWT_TOKEN == "":
  display.display(display.HTML("Please get JWT_TOKEN by visiting " +
                                 "<a href='" + SERVER_URL + "/login'>Login page</a>"))
  raise Exception("Please set JWT_TOKEN")
