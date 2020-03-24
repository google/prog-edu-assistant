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
      arguments = ["-c", "-f", tar_out.path, "--dereference", "--transform=s/^./autograder/", "-C", autograder_out.path, "."],
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

def _autograder_tar_impl(ctx):
  tar_inputs = [f for f in ctx.files.deps if f.path.endswith(".tar")]
  tar_paths = [f.path for f in tar_inputs]
  static_tar_paths = [f.path for f in ctx.files._static]
  binary_tar_paths = [f.path for f in ctx.files._binary]
  outs = []
  tarfile = ctx.label.name + ".tar"
  tar_out = ctx.actions.declare_file(tarfile)
  outs.append(tar_out)
  ctx.actions.run(
      inputs = tar_inputs + ctx.files._static + ctx.files._binary,
      outputs = [tar_out],
      progress_message = "Running tar %s" % tarfile,
      executable = "/usr/bin/tar",
      # Note 1: The below requires GNU tar.
      # Note 2: The resulting tar contains zero blocks, so needs -i option when extracting.
      arguments = (["--concatenate", "-f", tar_out.path] +
	tar_paths + static_tar_paths + binary_tar_paths),
  )
  return [DefaultInfo(files = depset(outs))]

# Defines a rule that concatenates autograder tar files for
# individual assignments and adds the static and binary files necessary
# for deployment.
autograder_tar = rule(
  implementation = _autograder_tar_impl,
  attrs = {
    "deps": attr.label_list(
	mandatory=True,
	allow_empty=False,
    ),
    "_static": attr.label(
	# Include the static files. This attribute should not be set by the user.
	default = Label("//static:static_tar"),
	cfg = "target",
    ),
    "_binary": attr.label(
	# Include the binary files. This attribute should not be set by the user.
	default = Label("//go:binary_tar"),
	cfg = "target",
    ),
  }
)

def _student_tar_impl(ctx):
  # Root prefix that notebook input files will have.
  prefix = ctx.bin_dir.path + '/' + ctx.build_file_path[:-len("/BUILD.bazel")]
  notebook_inputs = [f for f in ctx.files.srcs if f.path.endswith(".ipynb")]
  notebook_paths = [f.path[len(prefix)+1:] for f in notebook_inputs]
  outs = []
  tarfile = ctx.label.name + ".tar"
  tar_out = ctx.actions.declare_file(tarfile)
  outs.append(tar_out)
  ctx.actions.run(
      inputs = notebook_inputs,
      outputs = [tar_out],
      progress_message = "Running tar %s" % tarfile,
      executable = "/usr/bin/tar",
      arguments = (["-c", "-f", tar_out.path, "-C", prefix, "--dereference"] + notebook_paths),
  )
  return [DefaultInfo(files = depset(outs))]

# Defines a rule that collects all student notebooks into a tar file.
student_tar = rule(
  implementation = _student_tar_impl,
  attrs = {
    # The list of assignment_notebook target labels.
    "srcs": attr.label_list(
	mandatory=True,
	allow_empty=False,
    ),
  },
)
