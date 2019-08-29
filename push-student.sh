#!/bin/bash

set -ve

bash -ve ./build-student.sh

cd tmp/student
rm -rf helloworld* oop*
git init
git add -f .
git commit -a -m 'Student notebooks'
git push -f git@github.com:salikh/student-notebooks.git
