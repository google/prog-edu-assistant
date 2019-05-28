// Binary grade runs the autograder manually without a daemon.
// It assumes that the scratch directory has been alread set up:
// - autograder scripts files copied;
// - submission.py and submission_source.py created.
//
// Usage:
//
//   go run cmd/grade/grade.go
//     --autograder_dir ./autograder-scratch-dir
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/google/prog-edu-assistant/autograder"
)

var (
	autograderDir = flag.String("autograder_dir", "tmp",
		"The root directory of autograder scripts.")
	nsjailPath = flag.String("nsjail_path", "/usr/local/bin/nsjail",
		"The path to nsjail.")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
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
	ag.NSJailPath = *nsjailPath
	for _, filename := range flag.Args() {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("error reading %q: %s", filename, err)
		}
		report, err := ag.Grade(b)
		if err != nil {
			return fmt.Errorf("error grading %q: %s", filename, err)
		}
		fmt.Println(string(report))
	}
	return nil
}
