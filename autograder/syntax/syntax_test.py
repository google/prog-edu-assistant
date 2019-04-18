#!/usr/bin/env python3
"""A demo of checking a submitted solution from syntactic point of view."""

import ast
import json
import os
import re
import sys
import unittest

def readFile(filename):
    fullpath = os.path.join(os.path.dirname(__file__), filename)
    try:
        with open(fullpath) as f:
            source = f.read()
            return source
    except Exception as e:
        raise IOError("Could not open input file %s: %s" % (fullpath, e))

class BaseTest:
    """A base class to hold the test case logic.

    The tests will not be run for BaseTest, because it does not
    inherit from unittest.TestCase.
    """
    
    def test_parse(self):
        # Expect parse to succeed.
        tree = ast.parse(self.source)

    def test_has_print(self):
        # Expect source to have 'print'.
        regex = re.compile("print")
        if not regex.search(self.source):
            raise Exception("The solution does not have print")

    def test_has_hello(self):
        regex = re.compile('"Hello')
        if not regex.search(self.source):
            raise Exception("The solution does not have \"Hello\" string")


class Test1(unittest.TestCase, BaseTest):
    def setUp(self):
        self.source = readFile("submission1.py")

class Test2(unittest.TestCase, BaseTest):
    def setUp(self):
        self.source = readFile("submission2.py")

class Test3(unittest.TestCase, BaseTest):
    def setUp(self):
        self.source = readFile("submission3.py")

class Test4(unittest.TestCase, BaseTest):
    def setUp(self):
        self.source = readFile("submission4.py")

class Test5(unittest.TestCase, BaseTest):
    def setUp(self):
        self.source = readFile("submission5.py")

if __name__ == '__main__':
    unittest.main()
