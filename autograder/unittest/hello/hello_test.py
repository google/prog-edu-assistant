import unittest
import submission

class HelloTest(unittest.TestCase):
    def test_hello(self):
        self.assertEqual(submission.hello("world"), "Hello, world")

if __name__ == '__main__':
    unittest.main()
