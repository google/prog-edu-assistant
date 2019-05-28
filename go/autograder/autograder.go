// Package autograder provides the logic to parse the Jupyter notebook submissions,
// extract the assignment ID, match the assignment to the autograder scripts,
// set up the scratch directory and run the autograder tests under nsjail.
package autograder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/golang/glog"
	"github.com/google/prog-edu-assistant/notebook"
)

// Autograder encapsulates the setup of autograder scripts.
type Autograder struct {
	// Dir points to the root directory of autograder scripts.
	// Under Dir, the first level directory names are matched to assignment_id,
	// second level to exercise_id. In the second-level directories,
	// python unit test files should be present.
	Dir string
	// ScratchDir points to the directory where one can write, /tmp by default.
	ScratchDir string
	// NSJailPath is the path to nsjail, /usr/local/bin/nsjail by default.
	NSJailPath string
	// PythonPath is the path to python binary, /usr/bin/python by default.
	PythonPath string
	// DisableCleanup instructs the autograder not to delete the scratch directory.
	DisableCleanup bool
}

// New creates a new autograder instance given the root directory.
func New(dir string) *Autograder {
	return &Autograder{
		Dir:        dir,
		ScratchDir: "/tmp",
		NSJailPath: "/usr/local/bin/nsjail",
		PythonPath: "/usr/bin/python",
	}
}

// Grade takes a byte blob, tries to parse it as JSON, then tries to extract
// the metadata and match it to the available corpus of autograder scripts.
// If found, it then proceeds to run all autograder scripts under nsjail,
// parse the output, and produce the report, also in JSON format.
func (ag *Autograder) Grade(notebookBytes []byte) ([]byte, error) {
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
		return nil, fmt.Errorf("metadata does not have assignment_id")
	}
	assignmentID, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("metadata.assignment_id is not a string but %s",
			reflect.TypeOf(v))
	}
	dir := filepath.Join(ag.Dir, assignmentID)
	glog.V(3).Infof("dir = %q", dir)
	fs, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("assignment with id %q does not exit", assignmentID)
	}
	if !fs.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", dir)
	}
	n, err := notebook.Parse(notebookBytes)
	if err != nil {
		return nil, err
	}
	allOutcomes := make(map[string]bool)
	allReports := make(map[string]string)
	allLogs := make(map[string]interface{})
	baseScratchDir := filepath.Join(ag.ScratchDir, submissionID)
	err = os.MkdirAll(baseScratchDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("error making dir %q: %s", baseScratchDir, err)
	}
	for _, cell := range n.Cells {
		if cell.Metadata == nil {
			continue
		}
		v, ok := cell.Metadata["exercise_id"]
		if !ok {
			continue
		}
		exerciseID, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("exercise_id is not a string but %s",
				reflect.TypeOf(v))
		}
		exerciseDir := filepath.Join(dir, exerciseID)
		fs, err = os.Stat(exerciseDir)
		if err != nil {
			return nil, fmt.Errorf("exercise with id %s/%s does not exit",
				assignmentID, exerciseID)
		}
		if !fs.IsDir() {
			return nil, fmt.Errorf("%q is not a directory", exerciseDir)
		}
		scratchDir := filepath.Join(baseScratchDir, exerciseID)
		err := CopyDirFiles(exerciseDir, scratchDir)
		if err != nil {
			return nil, fmt.Errorf("error copying autograder scripts from %q to %q: %s", exerciseDir, scratchDir, err)
		}
		// TODO(salikh): Implement proper scratch management with overlayfs.
		filename := filepath.Join(scratchDir, "submission.py")
		err = ioutil.WriteFile(filename, []byte(cell.Source), 0775)
		if err != nil {
			return nil, fmt.Errorf("error writing to %q: %s", filename, err)
		}
		filename = filepath.Join(scratchDir, "submission_source.py")
		err = ioutil.WriteFile(filename, []byte(`source = """`+cell.Source+`"""`), 0775)
		if err != nil {
			return nil, fmt.Errorf("error writing to %q: %s", filename, err)
		}
		glog.V(3).Infof("Running tests in directory %s", scratchDir)
		outcomes, logs, err := ag.RunUnitTests(scratchDir)
		if err != nil {
			return nil, fmt.Errorf("error running unit tests in %q: %s", exerciseDir, err)
		}
		// Small data for the report generation.
		data := map[string]interface{}{
			"results": outcomes,
			"logs":    logs,
		}
		report, err := ag.RenderReports(scratchDir, data)
		if err != nil {
			return nil, err
		}
		allReports[exerciseID] = string(report)
		allLogs[exerciseID] = logs
		for k, v := range outcomes {
			_, ok := allOutcomes[k]
			if ok {
				return nil, fmt.Errorf("duplicated unit test %q", k)
			}
			allOutcomes[k] = v
		}
	}
	if !ag.DisableCleanup {
		_ = os.RemoveAll(baseScratchDir)
	}
	result := make(map[string]interface{})
	result["assignment_id"] = assignmentID
	result["submission_id"] = submissionID
	result["outcomes"] = allOutcomes
	result["logs"] = allLogs
	result["reports"] = allReports
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing report json: %s", err)
	}
	return b, nil
}

// nsjail -Mo --time_limit 2 --max_cpus 1 --rlimit_as 700 -E LANG=en_US.UTF-8 --disable_proc --chroot / --cwd $PWD --user nobody --group nogroup --iface_no_lo -- /usr/bin/python3 -m unittest discover -v -p '*Test.py'

var outcomeRegex = regexp.MustCompile(`(test[a-zA-Z0-9_]*) \(([a-zA-Z0-9_-]+)\.([a-zA-Z0-9_]*)\) \.\.\. (ok|FAIL|ERROR)`)

func (ag *Autograder) RunUnitTests(dir string) (map[string]bool, map[string]string, error) {
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
	outcomes := make(map[string]bool)
	logs := make(map[string]string)
	for _, fs := range fss {
		filename := fs.Name()
		if !strings.HasSuffix(filename, "Test.py") {
			continue
		}
		cmd := exec.Command(ag.NSJailPath,
			"-Mo",
			"--disable_clone_newcgroup",
			"--disable_clone_newipc",
			"--disable_clone_newnet",
			"--disable_clone_newns",
			"--disable_clone_newpid",
			"--disable_clone_newuser",
			"--disable_clone_newuts",
			"--disable_no_new_privs",
			"--time_limit", "3",
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
			// Overall status was non-ok.
			outcomes[filename] = false
		} else {
			// The file run okay.
			outcomes[filename] = true
		}
		logs[filename] = string(out)
		// TODO(salikh): Implement a more robust way of reporting individual
		// test statuses from inside the test runner.
		mm := outcomeRegex.FindAllSubmatch(out, -1)
		if len(mm) == 0 {
			// Cannot find any individual test case outcomes.
			outcomes[filename] = false
			continue
		}
		for _, m := range mm {
			method := string(m[1])
			className := string(m[3])
			status := string(m[4])
			key := className + "." + method
			if status == "ok" {
				outcomes[key] = true
			} else {
				outcomes[key] = false
			}
		}
	}
	return outcomes, logs, nil
}

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
			return fmt.Errorf(" CopyDirFiles: copying dirs recursively not implemented (%s/%s)", src, fs.Name())
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
