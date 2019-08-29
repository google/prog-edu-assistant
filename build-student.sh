#!/bin/bash
#
# A convenience script that rebuilds all student notebooks
# and prepares a directory to push to student github repo.

cd "$(dirname "$0")"
set -ve

rm -rf tmp/
bazel build ...
tar xvf bazel-genfiles/exercises/tmp-autograder_tar.tar
tar xvf bazel-genfiles/exercises/tmp-student_notebooks_tar.tar

cp -v student/* tmp/student/
cp -rv nbextensions tmp/student/
perl -i -pe \
  's,http://localhost:8000/upload,https://prog-edu-assistant.salikh.info/upload,g' \
  tmp/student/nbextensions/upload_it/main.js
