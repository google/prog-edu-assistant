// Binary functest is used for functional testing of the autograder
// with the upload server. It uses the canonical solutions and other
// submissions from the master notebooks to test how autograder
// handles various inputs.
//
// TODO(salikh): Check expectations.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path"

	"github.com/golang/glog"
)

var (
	submissionNotebook = flag.String("submission_notebook", "",
		"The path to the submission notebook.")
	serverURL = flag.String("server_url", "http://localhost:8000/upload",
		"The URL of the upload server where autograder is running.")
	fileInputName = flag.String("file_input_name", "notebook",
		"The name of the <input type='file'> element in the upload form.")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		glog.Exit(err)
	}
}

func run() error {
	// TODO(salikh): Implement constructing a submitted notebook from pristine
	// student notebook and the submission snippet.
	if *submissionNotebook != "" {
		glog.Infof("Submitting full notebook %s", *submissionNotebook)
		content, err := ioutil.ReadFile(*submissionNotebook)
		if err != nil {
			return fmt.Errorf("error reading %s: %s", *submissionNotebook, err)
		}
		body := new(bytes.Buffer)
		w := multipart.NewWriter(body)
		basename := path.Base(*submissionNotebook)
		part, err := w.CreateFormFile(*fileInputName, basename)
		if err != nil {
			return fmt.Errorf("multipart.Writer.CreateFormFile(%q, %q) returned error: %s",
				*fileInputName, path.Base(*submissionNotebook))
		}
		_, err = part.Write(content)
		if err != nil {
			return fmt.Errorf("FormFile.Write(%d bytes) returned error: %s", len(content), err)
		}
		err = w.Close()
		if err != nil {
			return fmt.Errorf("multipart.Writer.Close() returned error: %s", err)
		}
		req, err := http.NewRequest("POST", *serverURL, body)
		if err != nil {
			return fmt.Errorf("http.NewRequest() returned error: %s", err)
		}
		req.Header["Content-Type"] = []string{"multipart/form-data; boundary=" + w.Boundary()}
		client := &http.Client{}
		glog.Infof("about to send %v", req)
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("Error sending request to %s: %s", *serverURL, err)
		}
		glog.Infof("status code %v", resp.StatusCode)
		glog.Info(resp.Header)
		defer resp.Body.Close()
		b := make([]byte, 1048576)
		n, err := resp.Body.Read(b)
		if errors.Is(err, io.EOF) {
			b = b[:n]
		} else if err != nil {
			return fmt.Errorf("http.Response.Body.Read() returned error: %s", err)
		}
		os.Stdout.Write(b)
		fmt.Println()
		return nil
	}
	return fmt.Errorf("You need to specify what to submit via command line flag: " +
		"--submission_notebook")
}
