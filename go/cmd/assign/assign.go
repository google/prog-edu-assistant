// Binary assign is a tool to produce student notebooks and extract autograder scripts
// from master notebooks.
//
// Usage:
//
//   go run cmd/assign/assign.go
//     -command student
//     -input ../exercies/helloworld-en-master.ipynb
//     -output ./helloworld-student.ipynb
//
//   go run cmd/assign/assign.go
//     -command autograder
//     -input ../exercies/helloworld-en-master.ipynb
//     -output ./autograder-dir
//
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/google/prog-edu-assistant/notebook"
)

var (
	command = flag.String("command", "", "The command to execute.")
	input   = flag.String("input", "",
		"The file name of the input master notebook.")
	output = flag.String("output", "",
		"The file name of the output. If empty, output is written to stdout.")
)

type commandDesc struct {
	Help string
	Func func() error
}

var commands = map[string]commandDesc{
	"parse":      commandDesc{"Try parsing the input", parseCommand},
	"student":    commandDesc{"Extract student notebook", studentCommand},
	"autograder": commandDesc{"Extract autograder scripts", autograderCommand},
}

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if *command == "" {
		fmt.Printf("List of known commands:\n")
		for name, cmd := range commands {
			fmt.Printf("  %s   \t%s\n", name, cmd.Help)
		}
		return fmt.Errorf("command is not specified with --command.")
	}
	cmd, ok := commands[*command]
	if !ok {
		return fmt.Errorf("command %q is not defined", *command)
	}
	return cmd.Func()
}

func parseCommand() error {
	n, err := notebook.ParseFile(*input)
	if err != nil {
		return err
	}
	fmt.Printf("%d cells\n", len(n.Cells))
	for _, cell := range n.Cells {
		fmt.Printf("%s: %s\n", cell.Type, cell.Source)
		fmt.Println("--")
	}
	fmt.Printf("nbformat %d minor %d\n", n.NBFormat, n.NBFormatMinor)
	return nil
}

func studentCommand() error {
	n, err := notebook.ParseFile(*input)
	if err != nil {
		return err
	}
	n, err = n.ToStudent()
	if err != nil {
		return err
	}
	b, err := n.Marshal()
	if err != nil {
		return fmt.Errorf("error serializing notebook: %s", err)
	}
	if *output == "" {
		_, err := os.Stdout.Write(b)
		return err
	}
	return ioutil.WriteFile(*output, b, 0775)
}

func autograderCommand() error {
	n, err := notebook.ParseFile(*input)
	if err != nil {
		return err
	}
	n, err = n.ToAutograder()
	if err != nil {
		return err
	}
	assignmentID := n.Metadata["assignment_id"].(string)
	if *output == "" {
		fmt.Print("## Dry run mode. Would generate the following files:\n\n")
		for _, cell := range n.Cells {
			exerciseID := cell.Metadata["exercise_id"].(string)
			filename := cell.Metadata["filename"].(string)
			source := cell.Source
			fmt.Printf("-- %s/%s/%s:\n%s\n\n", assignmentID, exerciseID, filename, source)
		}
		return nil
	}
	err = os.MkdirAll(*output, 0775)
	if err != nil {
		return fmt.Errorf("could not create output directory %q: %s", *output, err)
	}
	for _, cell := range n.Cells {
		source := cell.Source
		filename := cell.Metadata["filename"].(string)
		exerciseID := cell.Metadata["exercise_id"].(string)
		dir := filepath.Join(*output, assignmentID, exerciseID)
		err = os.MkdirAll(dir, 0775)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(dir, filename), []byte(source), 0775)
	}
	return nil
}
