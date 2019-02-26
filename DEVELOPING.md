# Development of Programming Education Assistant

## Python

### Prerequisites
- Python3
- Python libraries: `absl-py`, `jupyter`, `yapf` and `pylint`.
  You can install them with `pip3 install <pkg> --user`.
- At this moment, libraries other than `jupyter` is used only for developments.

### Testing
In `python` directory, run

```shell
python3 -m utils_test
```

### Styleguide and lint
- Follow [Google Python Style Guide](http://google.github.io/styleguide/pyguide.html)
- Also, lint checks with `pylint` and code formatting with `yapf` are enforced.

### Validate your code
Before you send a PR, please make sure your code passes tests and lint checks.

```shell
python3 ./python/bin/validate.py
```
