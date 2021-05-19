"""A setup file for PIP pacakge.

Usage:

    python setup.py bdist_wheel
"""
import setuptools

with open('README.md', 'r') as f:
    LONG_DESCRIPTION = f.read()

setuptools.setup(
    name='prog_edu_assistant_tools',
    version='0.2',
    author='Salikh Zakirov',
    author_email='salikh@gmail.com',
    description=
    'Tools for authoring programming assignments in Jupyter notebooks',
    long_description=LONG_DESCRIPTION,
    long_description_content_type='text/markdown',
    url='https://github.com/google/prog-edu-assistant/tree/master/python/prog_edu_assistant_tools',
    packages=setuptools.find_packages(),
    classifiers=[
        'Programming Language :: Python :: 3',
        'License :: OSI Approved :: Apache Software License',
        'Operating System :: OS Independent',
    ],
    install_requires=[
        'IPython',
        'Jinja2',
    ],
)
