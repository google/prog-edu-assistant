import unittest
import submission

class InfLoopTest(unittest.TestCase):
    def test_infloop(self):
        # infloop() never returns
        self.assertEqual(submission.infloop(), 1)

if __name__ == '__main__':
    unittest.main()
