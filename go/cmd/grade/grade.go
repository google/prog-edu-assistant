// Binary grade runs the autograder manually without a daemon.
// It assumes that the scratch directory has been already set up:
// - autograder scripts files copied;
// - submission.py and submission_source.py created.
//
// Usage:
//
//   go run cmd/grade/grade.go
//     --autograder_dir ./autograder-scratch-dir
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"

	"github.com/google/prog-edu-assistant/autograder"
)

var (
	autograderDir = flag.String("autograder_dir", "",
		"The root directory of autograder scripts.")
	nsjailPath = flag.String("nsjail_path", "/usr/local/bin/nsjail",
		"The path to nsjail binary.")
	pythonPath = flag.String("python_path", "/usr/bin/python3",
		"The path to python binary.")
	scratchDir = flag.String("scratch_dir", "/tmp/autograde",
		"The base directory to create scratch directories for autograding.")
	disableCleanup = flag.Bool("disable_cleanup", false,
		"If true, does not delete scratch directory after running the tests.")
	autoRemove = flag.Bool("auto_remove", false,
		"If true, removes the scratch directory before creating a new one. "+
			"This is useful together with --disable_cleanup.")
	submissionID = flag.String("submission_id", "dummy",
		"The submission id.")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func setSubmissionID(b []byte, id string) ([]byte, error) {
	data := make(map[string]interface{})
	err := json.Unmarshal(b, &data)
	if err != nil {
		return nil, fmt.Errorf("could not parse request as JSON: %s", err)
	}
	v, ok := data["metadata"]
	if !ok {
		v = make(map[string]interface{})
		data["metadata"] = v
	}
	metadata, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("metadata is not a map, but %s", reflect.TypeOf(v))
	}
	metadata["submission_id"] = id
	return json.Marshal(data)
}

func run() error {
	if *autograderDir == "" {
		return fmt.Errorf("please specify --autograder_dir")
	}
	dir := *autograderDir
	if !filepath.IsAbs(dir) {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		dir = filepath.Join(cwd, dir)
	}
	dir = filepath.Clean(dir)
	ag := autograder.New(dir)
	ag.ScratchDir = *scratchDir
	ag.NSJailPath = *nsjailPath
	ag.PythonPath = *pythonPath
	ag.DisableCleanup = *disableCleanup
	ag.AutoRemove = *autoRemove
	for _, filename := range flag.Args() {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("error reading %q: %s", filename, err)
		}
		b, err = setSubmissionID(b, *submissionID)
		if err != nil {
			return fmt.Errorf("error setting submission_id: %s", err)
		}
		report, err := ag.Grade(b)
		if err != nil {
			return fmt.Errorf("error grading %q: %s", filename, err)
		}
		fmt.Println(string(report))
	}
	return nil
}
