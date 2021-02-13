# Running the tests inside of Colab

In this setup, the instructor notebook contains explanations and assignmends,
marked up canonical solutions and autochecking tests, as described in [../exercises/README.md].
To convert the instructor notebook into a student notebook, use the following command:

```
python python/colab/convert_to_student.py \
  --master_notebook ml-instructor.ipynb \
  --output_student_notebook ./ml-student-v5.ipynb
```

If you keep the instructor notebook on Colab too, there is handy script to download
a notebook for local processing:

```
python python/colab/gdrive_download.py \
  --file_id 1wMoaiqynAElDMxRdl3D2zrhS-gK5kOCF \
  --output_file ml-instructor.ipynb \
  --client_id_json_file ./client_id.json
```

The file `client_id.json` must contain a pair of OAuth client ID and client secret in JSON
format. You can use [Credentials](https://console.cloud.google.com/apis/credentials) section
in Google Cloud Console to create and download a client ID.

## Structure of the student notebook

The tests are stored inside of the metadata of the solution cells, together with `exercise_id`.

```
...
"metadata": {
  "id": "...",
  "exercise_id": "exercise_ml_0",
  "inlinetests": {
     "InlineTest_ml0": "\nassert 'num_training_examples' in globals()\n ...",
     ...
  },
  ...
}
```

To run self-checking tests, the student executes a cell, which contains a code piece like this:

```
# Run this cell to check your solution.
Check('exercise_ml_0')
```

The function Check obtains the full source of the notebook (ipynb JSON object), then
extracts the cell based on the passed `exercise_id`, by matching the passed string id
with `metadata["exercise_id"]`, and for the matching code cell, executes all inline tests.

Each inline tests is a pair of the name (key in the dictionary `inlinetests`) and the source
of the inline test (the value in the dictionary `inlinetests`).

The execution of the test happens in the same Python runtime that the notebook uses,
so all of the globals are available to the test. The tests typically 

## Preamble cell in the student notebook

To enable execution of self-checking tests, the student notebook needs to include a definition
of the function `Check()`. Please see the source code [preamble.py](../python/colab/preamble.py).

TODO(salikh): Move the definition of the `Check()` function into the PyPI
package `prog_edu_assistant_tools`.
