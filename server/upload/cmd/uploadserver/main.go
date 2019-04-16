package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"

	"github.com/google/prog-edu-assistant/uploadserver"
)

var (
	port        = flag.Int("port", 8000, "The port to serve HTTP/S.")
	useHTTPS    = flag.Bool("use_https", false, "If true, use HTTPS instead of HTTP.")
	sslCertFile = flag.String("ssl_cert_file", "localhost.crt",
		"The path to the signed SSL server certificate.")
	sslKeyFile = flag.String("ssl_key_file", "localhost.key",
		"The path to the SSL server key.")
	uploadDir   = flag.String("upload_dir", "uploads", "The directory to write uploaded notebooks.")
	disableCORS = flag.Bool("disable_cors", false, "If true, disables CORS browser checks. "+
		"This is currently necessary to enable uploads from Jupyter notebooks, but unfortunately "+
		"it also makes the server vulnerable to XSRF attacks. Use with care.")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	s := uploadserver.New(uploadserver.Options{
		UploadDir:   *uploadDir,
		DisableCORS: *disableCORS,
	})
	addr := ":" + strconv.Itoa(*port)
	protocol := "http"
	if *useHTTPS {
		protocol = "https"
	}
	fmt.Printf("\n  Serving on %s://localhost%s\n\n", protocol, addr)
	if *useHTTPS {
		return s.ListenAndServeTLS(addr, *sslCertFile, *sslKeyFile)
	}
	return s.ListenAndServe(addr)
}
