'''A sample module to test utils.TbFormatter.'''


def _divide_by_zero_child(val):
    return val / 0


def divide_by_zero():
    '''A function that throws ZeroDivisionError'''
    return _divide_by_zero_child(10)
