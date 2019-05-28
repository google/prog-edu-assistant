// Binary worker is the daemon that runs inside the autograder worker docker
// image, accepts requests on the message queue, runs autograder scripts
// under nsjail, creates reports and posts reports back to the message queue.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	"github.com/google/prog-edu-assistant/autograder"
	"github.com/google/prog-edu-assistant/queue"
)

var (
	queueSpec = flag.String("queue_spec", "amqp://guest:guest@localhost:5672/",
		"The spec of the queue to connect to.")
	autograderQueue = flag.String("autograder_queue", "autograde",
		"The name of the autograder queue to listen to the work requests.")
	reportQueue = flag.String("report_queue", "report",
		"The name of the queue to post the reports.")
	autograderDir = flag.String("autograder_dir", "",
		"The root directory of autograder scripts.")
	scratchDir = flag.String("scratch_dir", "/tmp",
		"The scratch directory, where one can write files.")
	nsjailPath = flag.String("nsjail_path", "/usr/local/bin/nsjail",
		"The path to nsjail binary.")
	pythonPath = flag.String("python_path", "/usr/bin/python3",
		"The path to python binary.")
	disableCleanup = flag.Bool("disable_cleanup", false,
		"If true, autograder will not delete scratch directory on success.")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if !filepath.IsAbs(*autograderDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		*autograderDir = filepath.Join(cwd, *autograderDir)
	}
	*autograderDir = filepath.Clean(*autograderDir)
	ag := autograder.New(*autograderDir)
	ag.NSJailPath = *nsjailPath
	ag.PythonPath = *pythonPath
	delay := 500 * time.Millisecond
	retryUntil := time.Now().Add(60 * time.Second)
	var q *queue.Channel
	var ch <-chan []byte
	for {
		var err error
		q, err = queue.Open(*queueSpec)
		if err != nil {
			if time.Now().After(retryUntil) {
				return fmt.Errorf("error opening queue %q: %s", *queueSpec, err)
			}
			glog.V(1).Infof("error opening queue %q: %s, retrying in %s", *queueSpec, err, delay)
			time.Sleep(delay)
			delay = delay * 2
			continue
		}
		ch, err = q.Receive(*autograderQueue)
		if err != nil {
			return fmt.Errorf("error receiving on queue %q: %s", *autograderQueue, err)
		}
		break
	}
	glog.Infof("Listening on the queue %q", *autograderQueue)
	// Enter the main work loop
	for b := range ch {
		glog.V(5).Infof("Received %d bytes: %s", len(b), string(b))
		report, err := ag.Grade(b)
		if err != nil {
			// TODO(salikh): Add remote logging and monitoring.
			log.Println(err)
		}
		glog.V(3).Infof("Grade result %d bytes: %s", len(report), string(report))
		err = q.Post(*reportQueue, report)
		if err != nil {
			log.Println(err)
		}
	}
	return nil
}
