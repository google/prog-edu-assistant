# Student notebook repo

This repository is not a true source repository. It is automatically generated
from https://github.com/salikh/prog-edu-assistant, which itself is a fork of
https://github.com/google/prog-edu-assistant.

## Binder

You can open notebooks from this repository by clicking
the following links:

* Functional programming/OOP:
https://mybinder.org/v2/gh/salikh/student-notebooks/master?filepath=functional-ja-student.ipynb
* Dataframes 1:
https://mybinder.org/v2/gh/salikh/student-notebooks/master?filepath=dataframe-pre1-ja-student.ipynb
* Dataframes 2:
https://mybinder.org/v2/gh/salikh/student-notebooks/master?filepath=dataframe-pre2-ja-student.ipynb
* Dataframes 3:
https://mybinder.org/v2/gh/salikh/student-notebooks/master?filepath=dataframe-pre3-ja-student.ipynb

## Local environment setup

### Conda

Use the following command to install the necessary package using Conda:

    conda install -c plotly plotly_express

### Virtualenv

If you use Debian-based Linux, use the following commands for local setup:

    apt-get install python-virtualenv
    virtualenv -p python3 ../venv
    source ../venv/bin/activate
    pip install -r requirements.txt

    jupyter nbextension install nbextensions/upload_it --user
    jupyter nbextension enable upload_it/main
    jupyter nbextensions_configurator enable --user

    jupyter notebook

## License

Copyright 2019 Google LLC.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
this repository except in compliance with the License. You may obtain a copy of
the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied. See the License for the
specific language governing permissions and limitations under the License.

See [LICENSE](LICENSE) for details.

## Disclaimer

This project is not an official Google project. It is not supported by Google
and Google specifically disclaims all warranties as to its quality,
merchantability, or fitness for a particular purpose.
