// Package autograder provides the logic to parse the Jupyter notebook submissions,
// extract the assignment ID, match the assignment to the autograder scripts,
// set up the scratch directory and run the autograder tests under nsjail.
package autograder

import (
	"bytes"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/golang/glog"
	"github.com/google/prog-edu-assistant/notebook"
	"github.com/sourcegraph/syntaxhighlight"
)

// Autograder encapsulates the setup of autograder scripts.
type Autograder struct {
	// Dir points to the root directory of autograder scripts.
	// Under Dir, the first level directory names are matched to assignment_id,
	// second level to exercise_id. In the second-level directories,
	// python unit test files (*Test.py) should be present.
	Dir string
	// ScratchDir points to the directory where one can write, /tmp by default.
	ScratchDir string
	// NSJailPath is the path to nsjail, /usr/local/bin/nsjail by default.
	NSJailPath string
	// PythonPath is the path to python binary, /usr/bin/python by default.
	PythonPath string
	// DisableCleanup instructs the autograder not to delete the scratch directory.
	DisableCleanup bool
	// AutoRemove instructs the autograder to delete the scratch directory path
	// before creating a new one. This is useful together with DisableCleanup.
	AutoRemove bool
	// IncludeLogs instructs the autograder to include the low-lever logs
	// from nsjail invocation into test report. This is useful for debugging.
	IncludeLogs bool
}

// New creates a new autograder instance given the autograder directory.
func New(dir string) *Autograder {
	return &Autograder{
		Dir:        dir,
		ScratchDir: "/tmp",
		NSJailPath: "/usr/local/bin/nsjail",
		PythonPath: "/usr/bin/python",
	}
}

type InlineTestFill struct {
	Context    string
	Submission string
	Inline     string
}

// The output format uses double braces to facilitate parsing
// of the output by regexps.
var inlineTestTmpl = template.Must(template.New("inlinetest").Parse(`import sys
{{if .Context}}
try:
  {{.Context}}
except Exception as e:
  print("\nWhile executing context: ERROR{{"{{"}}%s{{"}}"}}" % e)
  raise e
{{end}}
try:
  {{.Submission}}
except Exception as e:
  print("\nWhile executing submission: FAIL{{"{{"}}%s: %s{{"}}"}}" % (e.__class__, e))
  sys.exit(1)
try:
  {{.Inline}}
  print("OK{{"{{}}"}}")
except AssertionError as e:
  print("\nWhile executing inline test: FAIL{{"{{"}}%s{{"}}"}}" % str(e))
  sys.exit(1)
except Exception as e:
  print("\nWhile executing inline test: ERROR{{"{{"}}%s{{"}}"}}" % e)
  raise e
`))

func generateInlineTest(context, submission, test string) ([]byte, error) {
	var output bytes.Buffer
	context = strings.ReplaceAll(context, "\n", "\n  ")
	submission = strings.ReplaceAll(submission, "\n", "\n  ")
	test = strings.ReplaceAll(test, "\n", "\n  ")
	if strings.Trim(context, " \t\r\n") == "" {
		context = ""
	}
	err := inlineTestTmpl.Execute(&output, &InlineTestFill{
		// Indent the parts by two spaces to match the template.
		Context:    context,
		Submission: submission,
		Inline:     test,
	})
	if err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// CreateScratchDir takes the submitted contents of a solution cell,
// the source exercise directory and sets up the scratch directory
// for autograding.
func (ag *Autograder) CreateScratchDir(exerciseDir, scratchDir string, submission []byte) error {
	err := CopyDirFiles(exerciseDir, scratchDir)
	if err != nil {
		return fmt.Errorf("error copying autograder scripts from %q to %q: %s", exerciseDir, scratchDir, err)
	}
	// TODO(salikh): Implement proper scratch management with overlayfs.
	filename := filepath.Join(scratchDir, "submission.py")
	err = ioutil.WriteFile(filename, submission, 0644)
	if err != nil {
		return fmt.Errorf("error writing to %q: %s", filename, err)
	}
	filename = filepath.Join(scratchDir, "submission_source.py")
	sep := []byte{}
	if len(submission) > 0 && submission[len(submission)-1] == '"' {
		// If the submission ends in a quote, combination with triple quote would produce a syntax error,
		// so append a new line.
		sep = []byte("\n")
	}
	content := bytes.Join([][]byte{[]byte(`source = """`),
		bytes.ReplaceAll(submission, []byte(`"""`), []byte(`\"\"\"`)), sep, []byte(`"""`)}, nil)
	err = ioutil.WriteFile(filename, content, 0644)
	if err != nil {
		return fmt.Errorf("error writing to %q: %s", filename, err)
	}
	// Synthesize the inline tests.
	pattern := filepath.Join(exerciseDir, "*_inline.py")
	inlinetests, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("error in filepath.Glob(%q): %s", pattern, err)
	}
	for _, inlineTestFilename := range inlinetests {
		contextFilename := strings.ReplaceAll(inlineTestFilename, "_inline.py", "_context.py")
		contextContent, err := ioutil.ReadFile(contextFilename)
		if err != nil {
			return fmt.Errorf("error reading context file %q: %s", contextFilename, err)
		}
		testContent, err := ioutil.ReadFile(inlineTestFilename)
		if err != nil {
			return fmt.Errorf("error reading inline test file %q: %s", inlineTestFilename, err)
		}
		output, err := generateInlineTest(string(contextContent), string(submission), string(testContent))
		if err != nil {
			return fmt.Errorf("error generating inline test from template: %s", err)
		}
		outputFilename := filepath.Join(scratchDir,
			strings.ReplaceAll(filepath.Base(inlineTestFilename), "_inline.py", "_inlinetest.py"))
		err = ioutil.WriteFile(outputFilename, output, 0644)
		if err != nil {
			return fmt.Errorf("error writing the inline test file %q: %s", outputFilename, err)
		}
	}
	return nil
}

func joinInlineReports(inlineReports map[string]string) string {
	var names []string
	for name := range inlineReports {
		names = append(names, name)
	}
	sort.Strings(names)
	var parts []string
	for i, name := range names {
		report := inlineReports[name]
		//parts = append(parts, "<h4 style='color: #387;'>"+name+"</h4>")
		parts = append(parts, report)
		if i < len(names)-1 {
			parts = append(parts, "<br>")
		}
	}
	return strings.Join(parts, "\n")
}

type ErrorWithId struct {
	SubmissionID string
	Err          error
}

func (err *ErrorWithId) Error() string {
	return err.Err.Error()
}

func idErrorf(id string, f string, args ...interface{}) error {
	return &ErrorWithId{
		SubmissionID: id,
		Err:          fmt.Errorf(f, args...),
	}
}

// Grade takes a byte blob, tries to parse it as JSON, then tries to extract
// the metadata and match it to the available corpus of autograder scripts.
// If found, it then proceeds to run all autograder scripts under nsjail,
// parse the output, and produce the report, also in JSON format.
func (ag *Autograder) Grade(notebookBytes []byte) ([]byte, error) {
	glog.V(1).Infof("Grade notebook of %d bytes", len(notebookBytes))
	glog.V(5).Infof("Grade notebook:\n%s\n--", string(notebookBytes))
	data := make(map[string]interface{})
	err := json.Unmarshal(notebookBytes, &data)
	if err != nil {
		return nil, fmt.Errorf("could not parse request as JSON: %s", err)
	}
	v, ok := data["metadata"]
	if !ok {
		return nil, fmt.Errorf("request did not have .metadata")
	}
	metadata, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("metadata is not a map, but %s", reflect.TypeOf(v))
	}
	v, ok = metadata["submission_id"]
	if !ok {
		return nil, fmt.Errorf("request did not have submission_id")
	}
	submissionID, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("metadata.submission_id is not a string but %s",
			reflect.TypeOf(v))
	}
	v, ok = metadata["assignment_id"]
	if !ok {
		return nil, idErrorf(submissionID, "metadata does not have assignment_id")
	}
	assignmentID, ok := v.(string)
	if !ok {
		return nil, idErrorf(submissionID, "metadata.assignment_id is not a string but %s",
			reflect.TypeOf(v))
	}
	userHash := "unknown"
	v, ok = metadata["user_hash"]
	if ok {
		userHash, ok = v.(string)
		if !ok {
			return nil, idErrorf(submissionID, "metadata.user_hash is not a string but %s",
				reflect.TypeOf(v))
		}
	}
	var requestedExerciseID string
	if v, ok := metadata["requested_exercise_id"]; ok {
		val, ok := v.(string)
		if ok {
			requestedExerciseID = val
		}
	}
	dir := filepath.Join(ag.Dir, assignmentID)
	glog.V(3).Infof("assignment dir: %s", dir)
	fs, err := os.Stat(dir)
	if err != nil {
		return nil, idErrorf(submissionID, "assignment dir %q with id %q does not exit: %s", dir, assignmentID, err)
	}
	if !fs.IsDir() {
		return nil, idErrorf(submissionID, "%q is not a directory", dir)
	}
	n, err := notebook.Parse(notebookBytes)
	if err != nil {
		return nil, idErrorf(submissionID, "error parsing the submitted blob as Jupyter notebook: %s", err)
	}
	baseScratchDir := filepath.Join(ag.ScratchDir, submissionID)
	if ag.AutoRemove {
		// Remove the scratch dir if it exists.
		err = os.RemoveAll(baseScratchDir)
		if err != nil {
			return nil, idErrorf(submissionID, "error removing %q: %s", baseScratchDir, err)
		}
	} else {
		// Check that scratch dir does not exist.
		_, err = os.Stat(baseScratchDir)
		if err == nil {
			return nil, idErrorf(submissionID, "scratch dir %q already exists", baseScratchDir)
		}
	}
	err = os.MkdirAll(baseScratchDir, 0755)
	if err != nil {
		return nil, idErrorf(submissionID, "error making scratch dir %q: %s", baseScratchDir, err)
	}
	if !ag.DisableCleanup {
		defer func() {
			_ = os.RemoveAll(baseScratchDir)
		}()
	}
	result := make(map[string]interface{})
	exerciseFound := false
	for _, cell := range n.Cells {
		if cell.Metadata == nil {
			continue
		}
		v, ok := cell.Metadata["exercise_id"]
		if !ok {
			// Skip all non-solution cells.
			continue
		}
		exerciseID, ok := v.(string)
		if !ok {
			return nil, idErrorf(submissionID, "exercise_id is not a string but %s",
				reflect.TypeOf(v))
		}
		if requestedExerciseID != "" && requestedExerciseID != exerciseID {
			// Skip other exercises if requested a specific one.
			continue
		}
		exerciseFound = true
		exerciseDir := filepath.Join(dir, exerciseID)
		fs, err = os.Stat(exerciseDir)
		if err != nil {
			return nil, idErrorf(submissionID, "exercise with id %s/%s does not exit",
				assignmentID, exerciseID)
		}
		if !fs.IsDir() {
			return nil, idErrorf(submissionID, "%q is not a directory", exerciseDir)
		}
		glog.V(5).Infof("exercise_id: %s, source:\n%s\n--", exerciseID, cell.Source)
		scratchDir := filepath.Join(baseScratchDir, exerciseID)
		outcome, err := ag.GradeExercise(exerciseDir, scratchDir, cell.Source)
		if err != nil {
			return nil, idErrorf(submissionID, "error grading exercise %s: %s", exerciseID, err)
		}
		result[exerciseID] = outcome
	}
	if !exerciseFound {
		result["error"] = fmt.Sprintf("no exercises found. requested_exercise_id=%q", requestedExerciseID)
	}
	result["assignment_id"] = assignmentID
	result["user_hash"] = userHash
	result["submission_id"] = submissionID
	result["timestamp"] = time.Now().Unix()
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, idErrorf(submissionID, "error serializing report json: %s", err)
	}
	return b, nil
}

// GradeExercise grades one exercise given the read-only autograder directory
// for the exercise and the content of the submitted solution cell for the exercise.
// It sets up a scratch directory inside of the base scratch directory and runs
// all unit and inline tests. After running the tests, it looks for the templates
// in the directory and renders them. If there are no templates defined, it uses
// the autogenerated reports.
// Returns the outcome JSON object for the exercise, including the follwing fields:
// * logs: a map from test name to the merged test output, useful for debugging.
// * outcomes: a map from the test name to the test outcomes.
// * report: a raw HTML string containing all generated reports concatenated together.
//   Note, the order of the report concatenation is not well defined, so one is
//   expected to use only one template or only one inline test to get a predictable
//   output.
// Note: this function does not do any cleanup assuming that the caller will delete
// the base scratch directory.
func (ag *Autograder) GradeExercise(exerciseDir, scratchDir, submission string) (map[string]interface{}, error) {
	glog.V(3).Infof("Grade exercise %s, submission of %d bytes", exerciseDir, len(submission))
	glog.V(5).Infof("submission source:\n%s\n--", submission)
	// Check whether the submission is not trivial.
	filename := filepath.Join(exerciseDir, "empty_submission.py")
	if b, err := ioutil.ReadFile(filename); err == nil {
		if string(b) == submission {
			// The submission is not changed from the default state.
			exerciseName := filepath.Base(exerciseDir)
			return map[string]interface{}{
				"report": fmt.Sprintf("%s: empty submission", exerciseName),
			}, nil
		}
	}
	glog.Infof("exercise scratch dir: %s", scratchDir)
	err := ag.CreateScratchDir(exerciseDir, scratchDir, []byte(submission))
	if err != nil {
		return nil, fmt.Errorf("error creating scratch dir %s: %s", scratchDir, err)
	}
	glog.V(3).Infof("Running tests in directory %s", scratchDir)
	unitOutcomes, unitLogs, err := ag.RunUnitTests(scratchDir)
	if err != nil {
		return nil, fmt.Errorf("error running unit tests in %q: %s", scratchDir, err)
	}
	inlineOutcomes, inlineLogs, inlineReports, err := ag.RunInlineTests(scratchDir)
	if err != nil {
		return nil, fmt.Errorf("error running inline tests in %q: %s", scratchDir, err)
	}
	mergedOutcomes := make(map[string]interface{})
	mergedLogs := make(map[string]string)
	for k, v := range unitOutcomes {
		mergedOutcomes[k] = v
	}
	for k, v := range inlineOutcomes {
		mergedOutcomes[k] = v
	}
	for k, v := range unitLogs {
		mergedLogs[k] = v
	}
	for k, v := range inlineLogs {
		mergedLogs[k] = v
	}
	// The data object for the report generation.
	outcomeData := map[string]interface{}{
		"results": mergedOutcomes,
		"logs":    mergedLogs,
		"reports": inlineReports,
	}
	report, err := ag.RenderReports(scratchDir, outcomeData)
	if err != nil {
		return nil, err
	}
	if len(report) > 0 {
		// If there was a template, take its output, ignoring
		// autogenerated reports from inline tests.
		outcomeData["report"] = string(report)
	} else {
		// If there were no renderable template, or it returned an empty string,
		// Use the autogenerated reports from inline tests.
		outcomeData["report"] = joinInlineReports(inlineReports)
	}
	return outcomeData, nil
}

var outcomeRegex = regexp.MustCompile(`(test[a-zA-Z0-9_]*) \(([a-zA-Z0-9_-]+)\.([a-zA-Z0-9_]*)\) \.\.\. (ok|FAIL|ERROR)`)

// RunUnitTests runs all tests in a scratch directory found by a glob *Test.py.
// The name of the unit test is its base name without .py suffix.
func (ag *Autograder) RunUnitTests(dir string) (map[string]interface{}, map[string]string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting abs path for %q: %s", dir, err)
	}
	err = os.Chdir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("error on chdir %q: %s", dir, err)
	}
	fss, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("error on listing %q: %s", dir, err)
	}
	// outcomes is a map from test name to the object with the following fields:
	// * passed: boolean indicating whether the test run exited with 0 status (success).
	// * test_case_name: boolean indicating whether a specific test case passed or not.
	// Note, that if the there was an error during running the test, the outcome
	// may not contain all of the test case names, because the test case names
	// are extracted from the test runner logs.
	outcomes := make(map[string]interface{})
	// logs is a map from test name to the merged output.
	logs := make(map[string]string)
	for _, fs := range fss {
		filename := fs.Name()
		if !strings.HasSuffix(filename, "Test.py") {
			continue
		}
		// The test name is a file name with .py suffix stripped.
		testname := filename[:len(filename)-len(".py")]
		// nsjail -Mo --time_limit 2 --max_cpus 1 --rlimit_as 700 -E LANG=en_US.UTF-8 --disable_proc --chroot / --cwd $PWD --user nobody --group nogroup --iface_no_lo -- /usr/bin/python3 -m unittest discover -v -p '*Test.py'
		testOutcome := make(map[string]interface{})
		outcomes[testname] = testOutcome
		cmd := exec.Command(ag.NSJailPath,
			"-Mo",
			// NSJail does not work under docker without these disable flags.
			"--disable_clone_newcgroup",
			"--disable_clone_newipc",
			"--disable_clone_newnet",
			"--disable_clone_newns",
			"--disable_clone_newpid",
			"--disable_clone_newuser",
			"--disable_clone_newuts",
			"--disable_no_new_privs",
			"--time_limit", "30",
			"--max_cpus", "1",
			"--rlimit_as", "700",
			"--env", "LANG=en_US.UTF-8",
			"--disable_proc",
			//"--chroot", "/",
			"--cwd", dir,
			"--user", "nobody",
			"--group", "nogroup",
			"--iface_no_lo",
			"--",
			ag.PythonPath, "-m", "unittest",
			"-v", fs.Name())
		glog.V(5).Infof("about to execute %s %q", cmd.Path, cmd.Args)
		out, err := cmd.CombinedOutput()
		if err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				return nil, nil, fmt.Errorf("error running unit test command %q %q: %s", cmd.Path, cmd.Args, err)
			}
			// Overall there was an error running the test, or a failed test case.
			testOutcome["passed"] = false
		} else {
			// The test run with exit status 0 (success).
			testOutcome["passed"] = true
		}
		logs[filename] = string(out)
		// TODO(salikh): Implement a more robust way of reporting individual
		// test statuses from inside the test runner.
		mm := outcomeRegex.FindAllSubmatch(out, -1)
		if len(mm) == 0 {
			// Cannot find any individual test case outcomes, mark overall test as
			// not passed.
			testOutcome["passed"] = false
			testOutcome["error"] = "no test cases found"
			continue
		}
		for _, m := range mm {
			method := string(m[1])
			//className := string(m[3])  // Not used, as it is the same as testname.
			status := string(m[4])
			if status == "ok" {
				testOutcome[method] = true
			} else {
				testOutcome[method] = false
				testOutcome["passed"] = false
			}
		}
	}
	return outcomes, logs, nil
}

var (
	inlineOutcomeRegex = regexp.MustCompile(`(OK|ERROR|FAIL){{((?:[^}]|}[^}])*)}}`)
	syntaxErrorRegex   = regexp.MustCompile(`(?m)(SyntaxError: .*)$`)
	timeoutRegex       = regexp.MustCompile(`time limit.*Killing it`)
)

type inlineReportFill struct {
	FormattedSource htmltemplate.HTML
	Passed          bool
	Error           string
	Logs            string
}

// The template to render source code if syntax highlighter failed. It escapes the passed
// string and wraps it with <pre> element.
var sourceTmpl = htmltemplate.Must(htmltemplate.New("rawsource").Parse(`<pre>{{.}}</pre>`))

// The template to render reports from inline tests.
var inlineReportTmpl = htmltemplate.Must(htmltemplate.New("inlinereport").Parse(
	`<div class='code'>{{.FormattedSource}}</div>
{{if .Passed}}
<span class='ico green'>&check;</span><span class='message'>Looks OK.</span>
{{else}}
<span class='ico red'>&#x274C;</span><span class='message error'>{{.Error}}</span>
{{if .Logs}}
<h2>Logs</h2>
<div class='logs'>
{{.Logs}}
</div>
{{end}}
{{end}}
`))

// RunInlineTest runs the inline test specified by the filename (with
// user-sumitted code already written into the file surrounded by the context
// code and test code appropriately). It assumes the filename has the form of
// TestName_inlinetest.py.
// Returns the outcome JSON object with the following fields:
// * passed: a boolean indicating whether the test has passed.
// * error: if the test failed, a human-readable message explaining the error.
// Also returns the complete merged log of the test execution, as well
// as an autogenerated report for this inline test.
func (ag *Autograder) RunInlineTest(dir, filename, submissionFilename string) (map[string]interface{}, string, string, error) {
	submission, err := ioutil.ReadFile(submissionFilename)
	if err != nil {
		return nil, "", "", fmt.Errorf("error reading submission file %q: %s", submissionFilename, err)
	}
	outcome := make(map[string]interface{})
	cmd := exec.Command(ag.NSJailPath,
		"-Mo",
		// NSJail does not work under docker without these disable flags.
		"--disable_clone_newcgroup",
		"--disable_clone_newipc",
		"--disable_clone_newnet",
		"--disable_clone_newns",
		"--disable_clone_newpid",
		"--disable_clone_newuser",
		"--disable_clone_newuts",
		"--disable_no_new_privs",
		"--time_limit", "10",
		"--max_cpus", "1",
		"--rlimit_as", "700",
		"--env", "LANG=en_US.UTF-8",
		"--disable_proc",
		//"--chroot", "/",
		"--cwd", dir,
		"--user", "nobody",
		"--group", "nogroup",
		"--iface_no_lo",
		"--",
		ag.PythonPath,
		filename)
	var passed bool
	glog.V(5).Infof("about to execute %s %q", cmd.Path, cmd.Args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return nil, "", "", fmt.Errorf("error running unit test command %q %q: %s", cmd.Path, cmd.Args, err)
		}
		// Overall status was non-ok.
		passed = false
	} else {
		// The file was run successfully.
		passed = true
	}
	mm := syntaxErrorRegex.FindAllSubmatch(out, -1)
	if len(mm) > 0 {
		passed = false
		var parts []string
		for _, m := range mm {
			parts = append(parts, string(m[1]))
		}
		outcome["error"] = strings.Join(parts, "; ")
	}
	if timeoutRegex.Find(out) != nil {
		passed = false
		outcome["error"] = "Time out."
	}
	outcome["passed"] = passed
	mm = inlineOutcomeRegex.FindAllSubmatch(out, -1)
	if len(mm) == 0 {
		// Cannot find any individual test case outcomes.
		outcome["passed"] = false
	}
	var reportBuf bytes.Buffer
	for _, m := range mm {
		status := string(m[1])
		message := string(m[2])
		if status != "OK" {
			outcome["passed"] = false
		}
		if status == "ERROR" {
			message = "Test error: " + message
		}
		if message != "" {
			if old, ok := outcome["error"]; ok {
				outcome["error"] = old.(string) + "; " + message
			} else {
				outcome["error"] = message
			}
		}
	}
	formattedSource, err := syntaxhighlight.AsHTML(submission, syntaxhighlight.OrderedList())
	if err != nil {
		var sourceBuf bytes.Buffer
		err := sourceTmpl.Execute(&sourceBuf, submission)
		if err != nil {
			return nil, "", "", err
		}
		formattedSource = sourceBuf.Bytes()
	}
	message, _ := outcome["error"].(string)
	logs := ""
	if ag.IncludeLogs {
		logs = string(out)
	}
	err = inlineReportTmpl.Execute(&reportBuf, &inlineReportFill{
		FormattedSource: htmltemplate.HTML(formattedSource),
		Passed:          passed,
		Error:           message,
		Logs:            logs,
	})
	if err != nil {
		return nil, "", "", err
	}
	return outcome, string(out), reportBuf.String(), nil
}

// RunInlineTests runs all inline tests in a scratch directory found by a glob
// *_inlinetest.py.
// Returns
// - outcomes map[string]interface{}
// - logs map[string]string
// - reports map[string]string
func (ag *Autograder) RunInlineTests(dir string) (map[string]interface{}, map[string]string, map[string]string, error) {
	glog.V(3).Infof("RunInlineTests(%s)", dir)
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error getting abs path for %q: %s", dir, err)
	}
	err = os.Chdir(dir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error on chdir %q: %s", dir, err)
	}
	submissionFilename := filepath.Join(dir, "submission.py")
	_, err = os.Stat(submissionFilename)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error statting %s/submission.py: %s", dir, err)
	}
	fss, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error on listing %q: %s", dir, err)
	}
	outcomes := make(map[string]interface{})
	reports := make(map[string]string)
	logs := make(map[string]string)
	for _, fs := range fss {
		filename := fs.Name()
		if !strings.HasSuffix(filename, "_inlinetest.py") {
			continue
		}
		// Extract the test name by stripping _inlinetest.py.
		testname := filename[:len(filename)-len("_inlinetest.py")]
		testOutcome, testLog, testReport, err := ag.RunInlineTest(dir, filename, submissionFilename)
		if err != nil {
			return nil, nil, nil, err
		}
		outcomes[testname] = testOutcome
		logs[testname] = testLog
		if testReport != "" {
			reports[testname] = testReport
		}
	}
	return outcomes, logs, reports, nil
}

// RenderReports looks for report templates in the specified scratch dir and renders all reports.
// It returns the concatenation of all reports output.
func (ag *Autograder) RenderReports(dir string, data map[string]interface{}) ([]byte, error) {
	err := os.Chdir(dir)
	if err != nil {
		return nil, fmt.Errorf("error on chdir %q: %s", dir, err)
	}
	fss, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("error on listing %q: %s", dir, err)
	}
	dataJson, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var reports [][]byte
	for _, fs := range fss {
		filename := fs.Name()
		if !strings.HasSuffix(filename, "_template.py") {
			continue
		}
		cmd := exec.Command("python", filename)
		glog.V(3).Infof("Starting command %s %q with input %q", cmd.Path, cmd.Args, string(dataJson))
		cmdIn, err := cmd.StdinPipe()
		if err != nil {
			return nil, err
		}
		go func() {
			cmdIn.Write(dataJson)
			cmdIn.Close()
		}()
		output, err := cmd.CombinedOutput()
		if err != nil {
			reports = append(reports, []byte(fmt.Sprintf(`
<h2 style='color: red'>Reporter error</h2>
<pre>%s</pre>`, err.Error())))
			glog.Errorf("Reporter error: %s", err)
			reports = append(reports, output)
			continue
		}
		glog.V(3).Infof("Output: %s", string(output))
		reports = append(reports, output)
	}
	return bytes.Join(reports, nil), nil
}

// CopyDirFiles copies all files in the directory (one level).
func CopyDirFiles(src, dest string) error {
	err := os.MkdirAll(dest, 0755)
	if err != nil {
		return fmt.Errorf("error creating dir %q: %s", dest, err)
	}
	fss, err := ioutil.ReadDir(src)
	if err != nil {
		return fmt.Errorf("error listing dir %q: %s", src, err)
	}
	for _, fs := range fss {
		if fs.IsDir() {
			// Symlink the directory into the scratch directory.
			err := os.Symlink(filepath.Join(src, fs.Name()), filepath.Join(dest, fs.Name()))
			if err != nil {
				return fmt.Errorf("CopyDirFiles: error symlinking %s to %s: %s", fs.Name(), dest, err)
			}
			continue
		}
		filename := filepath.Join(src, fs.Name())
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("error reading %q: %s", filename, err)
		}
		filename = filepath.Join(dest, fs.Name())
		err = ioutil.WriteFile(filename, b, 0644)
		if err != nil {
			return fmt.Errorf("error writing %q: %s", filename, err)
		}
		glog.V(5).Infof("copied %s from %s to %s", fs.Name(), src, dest)
	}
	return nil
}
