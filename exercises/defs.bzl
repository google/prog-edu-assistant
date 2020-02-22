# TODO(salikh): Implement the automatic tar rules too
def assignment_notebook_macro(
	name,
	srcs,
	language = None,
	visibility = ["//visibility:private"]):
    """
    Defines a rule for student notebook and autograder
    generation from a master notebook.

    Arguments:
    name:
    srcs: the file name of the input notebook should end in '-master.ipynb'.
    """
    language_opt = ""
    if language:
      language_opt = " --language=" + language
    native.genrule(
	name = name + "_student",
	srcs = srcs,
	outs = [name + '-student.ipynb'],
	cmd = """$(location //go/cmd/assign) --input="$<" --output="$@" --preamble=$(location //exercises:preamble.py) --command=student""" + language_opt,
	tools = [
	    "//go/cmd/assign",
	    "//exercises:preamble.py",
	],
    )
    autograder_output = name + '-autograder'
    native.genrule(
	name = name + "_autograder",
	srcs = srcs,
	outs = [autograder_output],
	cmd = """$(location //go/cmd/assign) --input="$<" --output="$@" --command=autograder""" + language_opt,
	tools = [
	    "//go/cmd/assign",
	],
    )
