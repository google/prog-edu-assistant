def _assignment_notebook_impl(ctx):
  #print("src = ", ctx.attr.src)
  #print("src.path = ", ctx.file.src.path)
  outs = []
  languages = ctx.attr.languages
  inputs = [ctx.file.src]
  preamble_opt = ""
  if ctx.file.preamble:
    preamble_opt = " --preamble='" + ctx.file.preamble.path + "'"
    inputs.append(ctx.file.preamble)
    if ctx.attr.preamble_metadata:
      preamble_opt = preamble_opt + " --preamble_metadata='" + ctx.attr.preamble_metadata + "'"
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
    check_cell_opt = ""
    if ctx.attr.check_cell_template:
      check_cell_opt = (" --insert_check_cell --check_cell_template='" +
			ctx.attr.check_cell_template + "'")
    #print(" command = " + ctx.executable._assign.path + " --command=student --input='" + ctx.file.src.path + "'" + " --output='" + out.path + "'" + language_opt + preamble_opt + check_cell_opt)
    ctx.actions.run_shell(
      inputs = inputs,
      outputs = [out],
      tools = [ctx.executable._assign],
      progress_message = "Generating %s" % out.path,
      command = ctx.executable._assign.path + " --command=student --input='" + ctx.file.src.path + "'" + " --output='" + out.path + "'" + language_opt + preamble_opt + check_cell_opt,
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
    # If present, specifies the preamble metadata string.
    "preamble_metadata": attr.string(default="", mandatory=False),
    # If non-empty, enables insertion of check cells according to the template.
    "check_cell_template": attr.string(default="", mandatory=False),
    # This is private attribute used to capture the dependency
    # on the assign tool.
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


def strip_prefix(s, prefix):
  if s.startswith(prefix):
    s = s[len(prefix):]
    if s.startswith('/'):
      s = s[1:]
  return s


def _student_tar_impl(ctx):
  # Root prefix that notebook input files will have.
  prefix = ctx.bin_dir.path + '/' + ctx.build_file_path[:-len("/BUILD.bazel")]
  notebook_inputs = [f for f in ctx.files.deps if f.path.endswith(".ipynb")]
  notebook_paths = [strip_prefix(f.path, prefix) for f in notebook_inputs]
  # data dependencies can be direct (source files to include into output .tar file)
  # or .tar archives (to concatenate into output .tar file).
  data_inputs = [f for f in ctx.files.data if not f.path.endswith(".tar")]
  data_paths = [f.path for f in data_inputs]
  tar_inputs = [f for f in ctx.files.data if f.path.endswith(".tar")]
  tar_paths = [f.path for f in tar_inputs]
  outs = []
  # The final output.
  tarfile = ctx.label.name + ".tar"
  tar_out = ctx.actions.declare_file(tarfile)
  outs.append(tar_out)
  if len(tar_inputs) > 0:
    # The intermediate output only with files.
    files_tarfile = ctx.label.name + ".files.tar"
    files_tar_out = ctx.actions.declare_file(files_tarfile)
    # There are tar inputs. Generate in two steps.
    # Step 1: collect all file inputs into an intermediate tar.
    #print("tar command: /usr/bin/tar -c -f " + files_tar_out.path + " --dereference " + " ".join(data_paths)+ " -C " + prefix + " ".join(notebook_paths))
    ctx.actions.run(
	inputs = notebook_inputs + data_inputs,
	outputs = [files_tar_out],
	progress_message = "Running tar %s" % files_tarfile,
	executable = "/usr/bin/tar",
	arguments = (["-c", "-f", files_tar_out.path, "--dereference"] + data_paths + ["-C", prefix] + notebook_paths),
    )
    #print("tar command: /usr/bin/tar --concatenate -f " + tar_out.path + " " + files_tar_out.path + " ".join(tar_paths))
    ctx.actions.run(
	inputs = [files_tar_out] + tar_inputs,
	outputs = [tar_out],
	progress_message = "Running tar %s" % tarfile,
	executable = "/usr/bin/tar",
	arguments = (["--concatenate", "-f", tar_out.path, files_tar_out.path] + tar_paths),
    )
  else:
    # No tar inputs, just generate the output tar file.
    #print("tar command: /usr/bin/tar -c -f " + tar_out.path + "-C" + prefix + "--dereference "+ " ".join(notebook_paths + data_paths))
    ctx.actions.run(
	inputs = notebook_inputs + data_inputs,
	outputs = [tar_out],
	progress_message = "Running tar %s" % tarfile,
	executable = "/usr/bin/tar",
	arguments = (["-c", "-f", tar_out.path, "--dereference"] + data_paths + ["-C", prefix] + notebook_paths),
    )
  return [DefaultInfo(files = depset(outs))]

# Defines a rule that collects all student notebooks into a tar file.
# TODO(salikh): Allow student_tar to additionally specify data dependencies
# to package.
student_tar = rule(
  implementation = _student_tar_impl,
  attrs = {
    # The list of assignment_notebook target labels.
    "deps": attr.label_list(
	mandatory=True,
	allow_empty=False,
    ),
    # The list of tar labels to concatenate as data.
    "data": attr.label_list(
	mandatory=False,
	allow_empty=True,
	allow_files=True,
    ),
  },
)
