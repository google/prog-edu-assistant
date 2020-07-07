import unittest
import magics

#from prog_edu_assistant_tools import magics

class TestCutPrompt(unittest.TestCase):
  
  def testCutPrompt_intact(self):
    self.assertEqual(magics.MyMagics.CutPrompt('abc'), 'abc')
    self.assertEqual(magics.MyMagics.CutPrompt('\nabc\ncde\n'), '\nabc\ncde\n')

  def testCutPrompt_solution_markers(self):
    self.assertEqual(magics.MyMagics.CutPrompt('''
    aaa
    # BEGIN SOLUTION  
    xxx
    # END SOLUTION
    bbb
    '''), '''
    aaa
    xxx
    bbb
    ''')
    self.assertEqual(magics.MyMagics.CutPrompt('''
    aaa
    # BEGIN SOLUTION  
    xxx
    # END SOLUTION
    bbb
      # BEGIN SOLUTION  
      yyy
      # END SOLUTION
    ccc
    '''), '''
    aaa
    xxx
    bbb
      yyy
    ccc
    ''')

  def testCutPrompt_prompt(self):
    self.assertEqual(magics.MyMagics.CutPrompt('''
    aaa
    """ # BEGIN PROMPT 
    xxx
    """ # END PROMPT 
    bbb
      """ # BEGIN PROMPT  
      yyy
      """ # END PROMPT
    ccc
    '''), '''
    aaa
    bbb
    ccc
    ''')

if __name__ == '__main__':
  unittest.main()
