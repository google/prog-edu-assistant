import unittest


class SummaryTestResult(unittest.TextTestResult):
    """A small extension of TextTestResult that also collects a map of test statuses.

    result.results is a map from test name (string) to boolean: True(passed) or
    False(failed or error)
  """

    separator1 = "=" * 70
    separator2 = "-" * 70

    def __init__(self, stream, descriptions, verbosity):
        super(unittest.TextTestResult, self).__init__(stream, descriptions,
                                                      verbosity)
        # A map of test name to True(passed) or False(failed or error)
        self.results = {}
        # Copied from TextTestResult.
        self.stream = stream
        self.showAll = verbosity > 1
        self.dots = verbosity == 1
        self.descriptions = descriptions

    def testName(self, test):
        """A helper function to format the test as a human-readable string.

        The format is TestClassName.test_method. This is similar
        to TextTestResult.getDescription(test), but uses different format.
        getDescription: 'test_one (__main__.HelloTest)'
        testName: 'HelloTest.test_one'
        """
        return unittest.util.strclass(test.__class__).replace(
            "__main__.", "") + "." + test._testMethodName

    def getDescription(self, test):
        doc_first_line = test.shortDescription()
        if self.descriptions and doc_first_line:
            return "\n".join((str(test), doc_first_line))
        else:
            return str(test)

    def startTest(self, test):
        super(unittest.TextTestResult, self).startTest(test)
        if self.showAll:
            self.stream.write(self.getDescription(test))
            self.stream.write(" ... ")
            self.stream.flush()

    def addSuccess(self, test):
        super(unittest.TextTestResult, self).addSuccess(test)
        if self.showAll:
            self.stream.writeln("ok")
        elif self.dots:
            self.stream.write(".")
            self.stream.flush()
        self.results[self.testName(test)] = True

    def addError(self, test, err):
        super(unittest.TextTestResult, self).addError(test, err)
        if self.showAll:
            self.stream.writeln("ERROR")
        elif self.dots:
            self.stream.write("E")
            self.stream.flush()
        self.results[self.testName(test)] = False

    def addFailure(self, test, err):
        super(unittest.TextTestResult, self).addFailure(test, err)
        if self.showAll:
            self.stream.writeln("FAIL")
        elif self.dots:
            self.stream.write("F")
            self.stream.flush()
        self.results[self.testName(test)] = False

    def addSkip(self, test, reason):
        super(unittest.TextTestResult, self).addSkip(test, reason)
        if self.showAll:
            self.stream.writeln("skipped {0!r}".format(reason))
        elif self.dots:
            self.stream.write("s")
            self.stream.flush()

    def addExpectedFailure(self, test, err):
        super(unittest.TextTestResult, self).addExpectedFailure(test, err)
        if self.showAll:
            self.stream.writeln("expected failure")
        elif self.dots:
            self.stream.write("x")
            self.stream.flush()

    def addUnexpectedSuccess(self, test):
        super(unittest.TextTestResult, self).addUnexpectedSuccess(test)
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
