# pylint: disable=missing-docstring
"""Unit tests for utils.

This test uses files in testdata to check the results of TbFormatter.
To update them, run this test with --update flag.

python3 -m utils_test --update
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import os
import sys

from absl.testing import absltest
from absl import flags

import utils

FLAGS = flags.FLAGS

flags.DEFINE_bool('update', False, 'Update golden files in testdata.')


class TestTbFormatter(absltest.TestCase):
    def check_file(self, path, content):
        if FLAGS.update:
            with open(path, 'w', encoding='utf8') as writer:
                writer.write(content)
            return
        with open(path, 'r', encoding='utf8') as reader:
            self.assertEqual(reader.read(), content)

    @staticmethod
    def replace_dir(dirname, content):
        absdir = os.path.abspath(dirname)
        content = content.replace(absdir, '${dir}')
        home = os.path.expanduser('~')
        if absdir.startswith(home):
            content = content.replace('~' + absdir[len(home):], '${dir}')
        return content

    def test_format_tb(self):
        fmt = utils.TbFormatter(color='NoColor', tb_offset=1)
        text = ''
        try:
            from testdata import zero_division
            zero_division.divide_by_zero()
        except ZeroDivisionError:
            text = fmt.format(*sys.exc_info())
            text = self.replace_dir(os.path.dirname(__file__), text)
        self.check_file('testdata/formatted_tb.txt', text)

    def test_syntax_tb(self):
        fmt = utils.TbFormatter(color='NoColor')
        text = ''
        try:
            # pylint: disable=exec-used
            exec('x = 10; y =10; def f;')
        except SyntaxError:
            text = fmt.format(*sys.exc_info())
            text = self.replace_dir(os.path.dirname(__file__), text)
        self.check_file('testdata/formatted_syntax_tb.txt', text)


if __name__ == '__main__':
    # https://abseil.io/docs/python/guides/testing
    absltest.main()
