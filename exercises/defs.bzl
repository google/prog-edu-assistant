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

def _assignment_notebook_impl(ctx):
  print("src = ", ctx.attr.src)
  print("src.path = ", ctx.file.src.path)
  outs = []
  languages = ctx.attr.languages
  inputs = [ctx.file.src]
  preamble_opt = ""
  if ctx.file.preamble:
    preamble_opt = " --preamble='" + ctx.file.preamble.path + "'"
    inputs.append(ctx.file.preamble)
  if len(languages) == 0:
    # Force the language-agnostic notebook generation by default.
    languages = [""]
  for lang in languages:
    outfile = ctx.label.name + ("-" + lang  if lang else "") + "-student.ipynb"
    out = ctx.actions.declare_file(outfile)
    outs.append(out)
    language_opt = ""
    if lang:
      language_opt = " -language='" + lang + "'"
    print(" command = " + ctx.executable._assign.path + " --command=student --input='" + ctx.file.src.path + "'" + " --output='" + out.path + "'" + language_opt + preamble_opt)
    ctx.actions.run_shell(
      inputs = inputs,
      outputs = [out],
      tools = [ctx.executable._assign],
      progress_message = "Generating %s" % out.path,
      command = ctx.executable._assign.path + " --command=student --input='" + ctx.file.src.path + "'" + " --output='" + out.path + "'" + language_opt + preamble_opt,
    )
  
  # TODO(salikh): Consider if we need to generate language-specific
  # autograder directories.
  autograder_dir = ctx.label.name + '-autograder'
  autograder_out = ctx.actions.declare_directory(autograder_dir)
  outs.append(autograder_out)
  ctx.actions.run_shell(
      inputs = [ctx.file.src],
      outputs = [autograder_out],
      tools = [ctx.executable._assign],
      progress_message = "Generating %s" % autograder_out.path,
      command = ctx.executable._assign.path + " --command=autograder --input='" + ctx.file.src.path + "'" + " --output='" + autograder_out.path + "'",
  )
  tarfile = ctx.label.name + "-autograder.tar"
  tar_out = ctx.actions.declare_file(tarfile)
  outs.append(tar_out)
  ctx.actions.run(
      inputs = [autograder_out],
      outputs = [tar_out],
      progress_message = "Running tar %s" % tarfile,
      executable = "/usr/bin/tar",
      # Note: The below requires GNU tar.
      arguments = ["-c", "-f", tar_out.path, "--transform=s/^./autograder/", "-C", autograder_out.path, "."],
  )
  return [DefaultInfo(files = depset(outs))]

# Defines a rule for student notebook and autograder
# generation from a master notebook.
#
# Arguments:
#   name:
assignment_notebook = rule(
  implementation = _assignment_notebook_impl,
  attrs = {
    # Specifies the list of languages to generate student notebooks.
    # If omitted, defaults to empty list, which means that a
    # single language-agnostic notebook will be generated.
    # It is also possible to generate language-agnostic notebook
    # (skipping filtering by language) by adding an empty string
    # value to languages.
    "languages": attr.string_list(default=[], mandatory=False),
    # The file name of the input notebook.
    "src": attr.label(
	mandatory=True,
	allow_single_file=True),
    # If present, specifies the label of the preamble file.
    "preamble": attr.label(
	default=None,
	mandatory=False,
        allow_single_file=True),
    "_assign": attr.label(
	default = Label("//go/cmd/assign"),
	allow_single_file = True,
	executable = True,
	cfg = "host",
    ),
  },
)

