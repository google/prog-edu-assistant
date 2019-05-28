import setuptools

with open('README.md', 'r') as f:
    long_description = f.read()

setuptools.setup(
    name='prog_edu_assistant_tools',
    version='0.1',
    author='Salikh Zakirov',
    author_email='salikh@gmail.com',
    description=
    'Tools for authoring programming assignments in Jupyter notebooks',
    long_description=long_description,
    long_description_content_type='text/markdown',
    url=
    'https://github.com/google/prog-edu-assistant/tree/master/python/notebook_tools',
    packages=setuptools.find_packages(),
    classifiers=[
        'Programming Language :: Python :: 3',
        'License :: OSI Approved :: Apache Software License',
        'Operating System :: OS Independent',
    ],
)
