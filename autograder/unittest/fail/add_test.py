import unittest
import submission

class AddTest(unittest.TestCase):
    def test_0_0(self):
        self.assertEqual(submission.add(0,0), 0)

    def test_0_1(self):
        self.assertEqual(submission.add(0,1), 1)
        
    def test_2_2(self):
        self.assertEqual(submission.add(2,2), 4)

    def test_3_3(self):
        self.assertEqual(submission.add(3,3), 6)

if __name__ == '__main__':
    unittest.main()
