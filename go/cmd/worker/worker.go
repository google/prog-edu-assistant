// Binary worker is the daemon that runs inside the autograder worker docker
// container, accepts requests on the message queue, runs autograder scripts
// under nsjail, creates reports and posts reports back to the message queue.
//
// Usage:
//   go run cmd/worker/worker.go
//     -autograder_dir ./autograder-dir
//     -scratch_dir /tmp/autograder
//
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
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
	autoRemove = flag.Bool("auto_remove", false,
		"If true, removes the scratch directory before creating a new one. "+
			"This is useful together with --disable_cleanup.")
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
	ag.ScratchDir = *scratchDir
	ag.DisableCleanup = *disableCleanup
	ag.AutoRemove = *autoRemove
	// Exponential backoff on connecting to the message queue.
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
		glog.Infof("Worker received %d bytes", len(b))
		glog.V(5).Infof("Received notebook:\n%s\n--", string(b))
		reportBytes, err := ag.Grade(b)
		if err != nil {
			// TODO(salikh): Add monitoring.
			log.Println(err)
			errId, ok := err.(*autograder.ErrorWithId)
			if !ok {
				continue
			}
			// Report the error back to the user.
			var buf bytes.Buffer
			err := errorTmpl.Execute(&buf, err.Error())
			if err != nil {
				log.Println(err)
				continue
			}
			reportJSON := map[string]interface{}{
				"submission_id": errId.SubmissionID,
				"Report": map[string]interface{}{
					"report": buf.String(),
				},
			}
			reportBytes, err := json.MarshalIndent(reportJSON, "", "  ")
			if err != nil {
				log.Println(err)
				continue
			}
			err = q.Post(*reportQueue, reportBytes)
			if err != nil {
				glog.Errorf("Error posting %d byte report to queue %q: %s",
					len(reportBytes), *reportQueue, err)
			}
			continue
		}
		glog.V(3).Infof("Grade result %d bytes: %s",
			len(reportBytes), string(reportBytes))
		err = q.Post(*reportQueue, reportBytes)
		if err != nil {
			glog.Errorf("Error posting %d byte report to queue %q: %s", len(reportBytes), *reportQueue, err)
		}
		glog.V(5).Infof("Posted %d bytes to queue %q", len(reportBytes), *reportQueue)
	}
	return nil
}

var errorTmpl = template.Must(template.New("errortemplate").Parse(`
<h2 style='color: red'>Checker Error</h2>
<pre>{{.}}</pre>`))
