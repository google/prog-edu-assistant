// Binary post is for sending notebooks for grading.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/google/prog-edu-assistant/queue"
)

var (
	queueSpec = flag.String("queue_spec", "amqp://guest:guest@localhost:5672/",
		"The spec of the queue to connect to.")
	autograderQueue = flag.String("autograder_queue", "autograde",
		"The name of the autograder queue to listen to the work requests.")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	q, err := queue.Open(*queueSpec)
	if err != nil {
		return fmt.Errorf("error opening queue %q: %s", *queueSpec, err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	for _, filename := range flag.Args() {
		if !filepath.IsAbs(filename) {
			filename = filepath.Join(cwd, filename)
		}
		filename = filepath.Clean(filename)
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("error reading %q: %s", filename, err)
		}
		err = q.Post(*autograderQueue, b)
		if err != nil {
			return fmt.Errorf("error posting to %q: %s", *autograderQueue, err)
		}
	}
	return nil
}
