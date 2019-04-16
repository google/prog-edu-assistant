"""utility libraries for prog-edu-assistant."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from IPython.core import ultratb


# pylint: disable=too-few-public-methods
class TbFormatter:
    '''TbFormatter formats tracebacks of exceptions using IPython formatter.'''

    def __init__(self, tb_offset=0, color='Neutral'):
        self.syntax_tb = ultratb.SyntaxTB(color_scheme=color)
        self.interactive_tb = ultratb.AutoFormattedTB(mode='Context',
                                                      color_scheme=color,
                                                      tb_offset=tb_offset)

    def format(self, etype, value, traceback):
        '''Formats the output of sys.exc_info.'''
        if etype == SyntaxError:
            return self.syntax_tb.stb2text(
                self.syntax_tb.structured_traceback(etype, value, []))
        return self.interactive_tb.stb2text(
            self.interactive_tb.structured_traceback(etype, value, traceback))
