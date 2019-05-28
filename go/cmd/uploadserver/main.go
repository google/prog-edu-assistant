package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/google/prog-edu-assistant/queue"
	"github.com/google/prog-edu-assistant/uploadserver"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	port        = flag.Int("port", 8000, "The port to serve HTTP/S.")
	useHTTPS    = flag.Bool("use_https", false, "If true, use HTTPS instead of HTTP.")
	sslCertFile = flag.String("ssl_cert_file", "localhost.crt",
		"The path to the signed SSL server certificate.")
	sslKeyFile = flag.String("ssl_key_file", "localhost.key",
		"The path to the SSL server key.")
	allowCORSOrigin = flag.String("allow_cors_origin", "",
		"If non-empty, allow cross-origin requests from the specified domain."+
			"This is currently necessary to enable uploads from Jupyter notebooks, "+
			"but unfortunately "+
			"it also makes the server vulnerable to XSRF attacks. Use with care.")
	useOpenID = flag.Bool("use_openid", false, "If true, use OpenID Connect authentication"+
		" provided by the issuer specified with --openid_issuer.")
	openIDIssuer = flag.String("openid_issuer", "https://accounts.google.com",
		"The URL of the OpenID Connect issuer. "+
			"/.well-known/openid-configuration will be "+
			"requested for detailed endpoint configuration. Defaults to Google.")
	allowedUsersFile = flag.String("allowed_users_file", "",
		"The file name of a text file with one user email per line.")
	uploadDir = flag.String("upload_dir", "uploads", "The directory to write uploaded notebooks.")
	queueSpec = flag.String("queue_spec", "amqp://guest:guest@localhost:5672/",
		"The spec of the queue to connect to.")
	autograderQueue = flag.String("autograder_queue", "autograde",
		"The name of the autograder queue to send work requests.")
	reportQueue = flag.String("report_queue", "report",
		"The name of the queue to listen for the reports.")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	endpoint := google.Endpoint
	var userinfoEndpoint string
	if *openIDIssuer != "" {
		wellKnownURL := *openIDIssuer + "/.well-known/openid-configuration"
		resp, err := http.Get(wellKnownURL)
		if err != nil {
			return fmt.Errorf("Error on GET %s: %s", wellKnownURL, err)
		}
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		data := make(map[string]interface{})
		err = json.Unmarshal(b, &data)
		if err != nil {
			return fmt.Errorf("Error parsing response from %s: %s", wellKnownURL, err)
		}
		// Override the authentication endpoint.
		auth_ep, ok := data["authorization_endpoint"].(string)
		if !ok {
			return fmt.Errorf("response from %s does not have 'authorization_endpoint' key", wellKnownURL)
		}
		token_ep, ok := data["token_endpoint"].(string)
		if !ok {
			return fmt.Errorf("response from %s does not have 'token_endpoint' key", wellKnownURL)
		}
		endpoint = oauth2.Endpoint{
			AuthURL:   auth_ep,
			TokenURL:  token_ep,
			AuthStyle: oauth2.AuthStyleInParams,
		}
		glog.Infof("auth endpoint: %#v", endpoint)
		userinfoEndpoint, ok = data["userinfo_endpoint"].(string)
		if !ok {
			return fmt.Errorf("response from %s does not have 'userinfo_endpoint' key", wellKnownURL)
		}
		glog.Infof("userinfo endpoint: %#v", userinfoEndpoint)
	}
	allowedUsers := make(map[string]bool)
	if *allowedUsersFile != "" {
		b, err := ioutil.ReadFile(*allowedUsersFile)
		if err != nil {
			return fmt.Errorf("error reading --allowed_users_file %q: %s", *allowedUsersFile, err)
		}
		for _, email := range strings.Split(string(b), "\n") {
			if email == "" {
				continue
			}
			allowedUsers[email] = true
		}
	}
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
		ch, err = q.Receive(*reportQueue)
		if err != nil {
			return fmt.Errorf("error receiving on queue %q: %s", *autograderQueue, err)
		}
		break
	}
	addr := ":" + strconv.Itoa(*port)
	protocol := "http"
	if *useHTTPS {
		protocol = "https"
	}
	serverURL := fmt.Sprintf("%s://localhost%s", protocol, addr)
	if os.Getenv("SERVER_URL") != "" {
		// Allow override from the environment.
		serverURL = os.Getenv("SERVER_URL")
	}
	s := uploadserver.New(uploadserver.Options{
		AllowCORSOrigin:  *allowCORSOrigin,
		ServerURL:        serverURL,
		UploadDir:        *uploadDir,
		Channel:          q,
		QueueName:        *autograderQueue,
		UseOpenID:        *useOpenID,
		AllowedUsers:     allowedUsers,
		AuthEndpoint:     endpoint,
		UserinfoEndpoint: userinfoEndpoint,
		ClientID:         os.Getenv("CLIENT_ID"),
		ClientSecret:     os.Getenv("CLIENT_SECRET"),
		CookieAuthKey:    os.Getenv("COOKIE_AUTH_KEY"),
		CookieEncryptKey: os.Getenv("COOKIE_ENCRYPT_KEY"),
	})
	fmt.Printf("\n  Serving on %s\n\n", serverURL)
	if *useHTTPS {
		return s.ListenAndServeTLS(addr, *sslCertFile, *sslKeyFile)
	}
	go s.ListenForReports(ch)
	return s.ListenAndServe(addr)
}
