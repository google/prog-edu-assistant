"""A package providing SummaryTestResult.

This package provides SummaryTestResult, which is an extension
of unittest.TextTestResult that also collects summary of test case
statuses: True (passed) / False (failed/error) in a dictionary
keyed by the 'TestClass.test_method'.
"""
import unittest


def test_name(test):
    """A helper function to format the test as a human-readable string.

    The format is TestClassName.test_method. This is similar
    to TextTestResult.getDescription(test), but uses different format.
    getDescription: 'test_one (__main__.HelloTest)'
    test_name: 'HelloTest.test_one'
    """
    return unittest.util.strclass(test.__class__).replace(
        "__main__.", "") + "." + test._testMethodName  # pylint: disable=W0212


def test_class_name(self, test):
    "A helper function to report the test class name."
    return unittest.util.strclass(test.__class__).replace(
        "__main__.", "")


def test_method_name(self, test):
    "A helper function to report the test case method name."
    return test._testMethodName


class SummaryTestResult(unittest.TextTestResult):
    """A small extension of TextTestResult that also collects a map of test statuses.

    result.results is a map from test class name (string) to the map of test
    case name (string) to boolean: True(passed) or False(failed or error).
  """

    separator1 = "=" * 70
    separator2 = "-" * 70

    def __init__(self, stream, descriptions, verbosity):
        super().__init__(stream, descriptions, verbosity)
        # A map of test name to True(passed) or False(failed or error)
        self.results = {}
        # Copied from TextTestResult.
        self.stream = stream
        self.showAll = verbosity > 1
        self.dots = verbosity == 1
        self.descriptions = descriptions

    def getDescription(self, test):
        doc_first_line = test.shortDescription()
        if self.descriptions and doc_first_line:
            return "\n".join((str(test), doc_first_line))
        return str(test)

    def startTest(self, test):
        super().startTest(test)
        if self.showAll:
            self.stream.write(self.getDescription(test))
            self.stream.write(" ... ")
            self.stream.flush()

    def addSuccess(self, test):
        super().addSuccess(test)
        if self.showAll:
            self.stream.writeln("ok")
        elif self.dots:
            self.stream.write(".")
            self.stream.flush()
        if test_class_name(test) not in self.results:
            self.results[test_class_name(test)] = {}
        self.results[test_class_name(test)][test_method_name(test)] = True
        if 'passed' not in self.results[test_class_name(test)]:
            self.results[test_class_name(test)]['passed'] = True

    def addError(self, test, err):
        super().addError(test, err)
        if self.showAll:
            self.stream.writeln("ERROR")
        elif self.dots:
            self.stream.write("E")
            self.stream.flush()
        if test_class_name(test) not in self.results:
            self.results[test_class_name(test)] = {}
        self.results[test_class_name(test)][test_method_name(test)] = False
        self.results[test_class_name(test)]['passed'] = False

    def addFailure(self, test, err):
        super().addFailure(test, err)
        if self.showAll:
            self.stream.writeln("FAIL")
        elif self.dots:
            self.stream.write("F")
            self.stream.flush()
        if test_class_name(test) not in self.results:
            self.results[test_class_name(test)] = {}
        self.results[test_class_name(test)][test_method_name(test)] = False
        self.results[test_class_name(test)]['passed'] = False

    def addSkip(self, test, reason):
        super().addSkip(test, reason)
        if self.showAll:
            self.stream.writeln("skipped {0!r}".format(reason))
        elif self.dots:
            self.stream.write("s")
            self.stream.flush()

    def addExpectedFailure(self, test, err):
        super().addExpectedFailure(test, err)
        if self.showAll:
            self.stream.writeln("expected failure")
        elif self.dots:
            self.stream.write("x")
            self.stream.flush()

    def addUnexpectedSuccess(self, test):
        super().addUnexpectedSuccess(test)
        if self.showAll:
            self.stream.writeln("unexpected success")
        elif self.dots:
            self.stream.write("u")
            self.stream.flush()

    def printErrors(self):
        if self.dots or self.showAll:
            self.stream.writeln()
        self.printErrorList("ERROR", self.errors)
        self.printErrorList("FAIL", self.failures)

    def printErrorList(self, flavour, errors):
        for test, err in errors:
            self.stream.writeln(self.separator1)
            self.stream.writeln("%s: %s" %
                                (flavour, self.getDescription(test)))
            self.stream.writeln(self.separator2)
            self.stream.writeln("%s" % err)
