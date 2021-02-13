# Local grading

NOTE: local grading is very experimental and may have substantial drawbacks
compared to running self-checking tests in Colab.

Once you have downloaded the student notebook submissions, you can run a script to run
auto-checker tests in grader mode to bulk-grade student work.

```
python python/colab/grade_student.py \
  --instructor_notebook ml-instructor.ipynb \
  --student_notebook ml-student-v5.ipynb \
  --noalsologtostderr \
  --context 'import tensorflow as tf; import numpy as np; import itertools; import tensorflow_datasets as tfds; ds, info = tfds.load("fashion_mnist", with_info=True)`
```

The scripts extracts solution cells from the student notebook and uses
functions from `convert_to_student.py` to extract the test snippets from the
instructor notebooks, and run them as inline tests.
Inline test is run in the following steps, where all code is evaluated inside the same Python
interpreter using `exec()` builtin function.

1. Run the context snippet provided on the command line
2. Run the student submission (solution cell only)
3. Run the inline test code

TODO(salikh): Improve the script `grade_student.py` to output the full test results.
