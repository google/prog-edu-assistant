// Package uploadserver provides an implemenation of the upload server
// that accepts student notebook uploads (submissions), posts them to the
// message queue for grading, listens for the reports on the message queue
// and makes reports available on the web.
package uploadserver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"time"

	"github.com/golang/glog"
	"github.com/google/prog-edu-assistant/queue"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

// Options configures the behavior of the web server.
type Options struct {
	// The base URL of this server. This is used to construct callback URL.
	ServerURL string
	// UploadDir specifies the directory to write uploaded files to
	// and to serve on /uploads.
	UploadDir string
	// AllowCORSOrigin specifies the origin allowed for cross-origin requests.
	// This value is returned in
	// Access-Control-Allow-Origin: HTTP header.
	AllowCORSOrigin string
	// QueueName is the name of the queue to post uploads.
	QueueName string
	// Channel is the interface to the message queue.
	*queue.Channel
	// UseOpenID enables authentication using OpenID Connect.
	UseOpenID bool
	// AllowedUsers lists the users that are authorized to use this service.
	// If the map is empty, no access control is performed, only authentication.
	AllowedUsers map[string]bool
	// AuthEndpoint specifies the OpenID Connect authentication and token endpoints.
	AuthEndpoint oauth2.Endpoint
	// UserinfoEndpoint specifies the user info endpoint.
	UserinfoEndpoint string
	// ClientID is used for OpenID Connect authentication.
	ClientID string
	// ClientSecret is used for OpenID Connect authentication.
	ClientSecret string
	// Set to 32 or 64 random bytes.
	CookieAuthKey string
	// Set to 16, 24 or 32 random bytes.
	CookieEncryptKey string
}

// Server provides an implementation of a web server for handling student
// notebook uploads.
type Server struct {
	opts            Options
	mux             *http.ServeMux
	reportTimestamp map[string]time.Time
	cookieStore     *sessions.CookieStore
	// OauthConfig specifies endpoing configuration for the OpenID Connect
	// authentication.
	oauthConfig *oauth2.Config
	// A random value used to match authentication callback to the request.
	oauthState string
}

// New creates a new Server instance.
func New(opts Options) *Server {
	mux := http.NewServeMux()
	s := &Server{
		opts:            opts,
		mux:             mux,
		reportTimestamp: make(map[string]time.Time),
		cookieStore:     sessions.NewCookieStore([]byte(opts.CookieAuthKey), []byte(opts.CookieEncryptKey)),
		oauthConfig: &oauth2.Config{
			RedirectURL:  opts.ServerURL + "/callback",
			ClientID:     opts.ClientID,
			ClientSecret: opts.ClientSecret,
			Scopes:       []string{"profile", "email", "openid"},
			Endpoint:     opts.AuthEndpoint,
		},
		oauthState: uuid.New().String(),
	}
	mux.Handle("/", handleError(s.uploadForm))
	mux.Handle("/upload", handleError(s.handleUpload))
	mux.Handle("/uploads/", http.StripPrefix("/uploads",
		http.FileServer(http.Dir(s.opts.UploadDir))))
	mux.HandleFunc("/favicon.ico", s.handleFavIcon)
	mux.Handle("/report/", handleError(s.handleReport))
	if s.opts.UseOpenID {
		mux.Handle("/login", handleError(s.handleLogin))
		mux.Handle("/callback", handleError(s.handleCallback))
		mux.Handle("/logout", handleError(s.handleLogout))
		mux.Handle("/profile", handleError(s.handleProfile))
	}
	return s
}

const UserSessionName = "user_session"

// ListenAndServe starts the server similarly to http.ListenAndServe.
func (s *Server) ListenAndServe(addr string) error {
	err := os.MkdirAll(s.TmpDir, 0700)
	if err != nil {
		return err
	}
	return http.ListenAndServe(addr, s.mux)
}

// ListenAndServeTLS starts a server using HTTPS.
func (s *Server) ListenAndServeTLS(addr, certFile, keyFile string) error {
	return http.ListenAndServeTLS(addr, certFile, keyFile, s.mux)
}

// httpError wraps the HTTP status code and makes them usable as Go errors.
type httpError int

func (e httpError) Error() string {
	return http.StatusText(int(e))
}

// handleError is a convenience wrapper that converts Go convention of returning
// an error into an HTTP error. This kind of reporting is not possible if the
// handler function has already written HTTP headers, but this rarely happens
// in practice, but makes development much more convenient.
//
// TODO(salikh): Reconsider error reporting in production deployment.
func handleError(fn func(http.ResponseWriter, *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		err := fn(w, req)
		if err != nil {
			glog.Errorf("%s  %s", req.URL, err.Error())
			status, ok := err.(httpError)
			if ok {
				http.Error(w, err.Error(), int(status))
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	})
}

func (s *Server) handleFavIcon(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "image/x-icon")
	w.Write(favIcon)
}

// handleReport gets the file name from the HTTP request URI path component,
// checks if the matching file (with .txt suffix) exists in the upload
// directory, and serves it if it exists. If the file does not exists, it
// serves a small piece of HTML with inline Javascript that automatically
// reloads itself with exponential backoff. After a few retries it reports
// generic error and stops autoreloading. Note that if the user manually refreshes
// the page later, the same autoreload process is repeated. This process
// is designed to handle the case of workers being overloaded with graing work
// and producing reports with long delay.
func (s *Server) handleReport(w http.ResponseWriter, req *http.Request) error {
	if s.opts.AllowCORSOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", s.opts.AllowCORSOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	basename := path.Base(req.URL.Path)
	filename := filepath.Join(s.opts.UploadDir, basename+".txt")
	glog.V(5).Infof("checking %q for existence", filename)
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		// Serve a placeholder autoreload page.
		reloadMs := int64(500)
		ts := s.reportTimestamp[basename]
		if ts.IsZero() {
			// Store the first request time
			s.reportTimestamp[basename] = time.Now()
			// TODO(salikh): Eventually clean up old entries from reportTimestamp map.
		} else {
			// Back off automatically.
			reloadMs = time.Since(ts).Nanoseconds() / 1000000
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// TODO(salikh): Make timeout configurable.
		if reloadMs > 10000 {
			// Reset for retry.
			reloadMs = 500
			s.reportTimestamp[basename] = time.Now()
		}
		if reloadMs > 5000 {
			fmt.Fprintf(w, `<title>Something weng wrong</title>
<h2>Error</h2>
Something went wrong, please retry your upload.
`)
			return nil
		}
		fmt.Fprintf(w, `<title>Please wait</title>
<script>
function refresh(t) {
	setTimeout("location.reload(true)", t)
}
</script>
<body onload="refresh(%d)">
<h2>Waiting for %d seconds, report is being generated now</h2>
</body>`, reloadMs, (reloadMs+999)/1000)
		return nil
	}
	if err != nil {
		return err
	}
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	data := make(map[string]interface{})
	err = json.Unmarshal(b, &data)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<title>Report for %s</title>`, basename)
	if v, ok := data["reports"]; ok {
		reports, ok := v.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected reports to be map[string]interface{}, got %s", reflect.TypeOf(v))
		}
		// Just concatenate all reports.
		for exercise_id, report := range reports {
			fmt.Fprintf(w, "<h2>%s</h2>", exercise_id)
			html, ok := report.(string)
			if !ok {
				return fmt.Errorf("expected report to be a string, got %s", reflect.TypeOf(report))
			}
			fmt.Fprint(w, html)
		}
	}
	return nil
}

// handleLogin handles Open ID Connect authentication.
func (s *Server) handleLogin(w http.ResponseWriter, req *http.Request) error {
	url := s.oauthConfig.AuthCodeURL(s.oauthState)
	http.Redirect(w, req, url, http.StatusTemporaryRedirect)
	return nil
}

// getUserInfo requests user profile by issuing an independent HTTP GET request
// using the authentication code received by the callback.
func (s *Server) getUserInfo(state string, code string) ([]byte, error) {
	if state != s.oauthState {
		return nil, fmt.Errorf("invalid oauth state")
	}
	token, err := s.oauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange failed: %s", err)
	}
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting user info: %s", err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading user info response: %s", err)
	}
	return b, nil
}

// UserProfile defines the fiels that Open ID Connect server may return in
// response to a profile request.
type UserProfile struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Link          string `json:"link"`
	Picture       string `json:"picture"`
}

// handleCallback handles the OAuth2 callback.
func (s *Server) handleCallback(w http.ResponseWriter, req *http.Request) error {
	req.ParseForm()
	b, err := s.getUserInfo(req.FormValue("state"), req.FormValue("code"))
	if err != nil {
		return err
	}
	var profile UserProfile
	//fmt.Printf("%s\n", b)
	err = json.Unmarshal(b, &profile)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	session, err := s.cookieStore.Get(req, UserSessionName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}
	if len(s.opts.AllowedUsers) > 0 && !s.opts.AllowedUsers[profile.Email] {
		delete(session.Values, "email")
		session.Save(req, w)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(fmt.Sprintf("<title>Forbidden</title>User %s is not authorized.<br>"+
			"Try a different Google account. <a href='https://mail.google.com/mail/logout'>Log out of Google</a>.", profile.Email)))
		return nil
	}
	session.Values["email"] = profile.Email
	session.Save(req, w)
	http.Redirect(w, req, "/profile", http.StatusTemporaryRedirect)
	return nil
}

// handleProfile reports the current authentication data. This is mostly
// useful for ad-hoc testing of authentication.
func (s *Server) handleProfile(w http.ResponseWriter, req *http.Request) error {
	session, err := s.cookieStore.Get(req, UserSessionName)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	email, ok := session.Values["email"]
	if ok {
		fmt.Fprintf(w, "Logged in as %s. <a href='/logout'>Log out link</a>.", email)
		fmt.Fprintf(w, "<p><strong>You can close this window and retry upload now.</strong>")
	} else {
		fmt.Fprintf(w, "Logged out. <a href='/login'>Log in</a>.")
	}
	return nil
}

// handleLogout clears the user cookie.
func (s *Server) handleLogout(w http.ResponseWriter, req *http.Request) error {
	session, err := s.cookieStore.Get(req, UserSessionName)
	if err != nil {
		return err
	}
	delete(session.Values, "email")
	session.Save(req, w)
	http.Redirect(w, req, "/profile", http.StatusTemporaryRedirect)
	return nil
}

// authenticate handles the authentication. If authentication or authorization
// was not successful, it returns an error.
func (s *Server) authenticate(w http.ResponseWriter, req *http.Request) error {
	session, err := s.cookieStore.Get(req, UserSessionName)
	if err != nil {
		return err
	}
	email, ok := session.Values["email"].(string)
	glog.V(3).Infof("authenticate %s: email=%s", req.URL, session.Values["email"])
	if !ok || email == "" {
		return httpError(http.StatusUnauthorized)
	}
	if len(s.opts.AllowedUsers) > 0 && !s.opts.AllowedUsers[email] {
		return httpError(http.StatusForbidden)
	}
	return nil
}

const maxUploadSize = 1048576

// handleUpload handles the upload requests via web form.
func (s *Server) handleUpload(w http.ResponseWriter, req *http.Request) error {
	if s.opts.AllowCORSOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", s.opts.AllowCORSOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	glog.Infof("%s %s", req.Method, req.URL.Path)
	if req.Method == "OPTIONS" {
		return nil
	}
	if s.opts.UseOpenID {
		err := s.authenticate(w, req)
		if err != nil {
			return err
		}
	}
	if req.Method != "POST" {
		return fmt.Errorf("Unsupported method %s on %s", req.Method, req.URL.Path)
	}
	req.Body = http.MaxBytesReader(w, req.Body, maxUploadSize)
	err := req.ParseMultipartForm(maxUploadSize)
	if err != nil {
		return fmt.Errorf("error parsing upload form: %s", err)
	}
	f, _, err := req.FormFile("notebook")
	if err != nil {
		return fmt.Errorf("no notebook file in the form: %s\nRequest %s", err, req.URL)
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("error reading upload: %s", err)
	}
	// TODO(salikh): Add user identifier to the file name.
	submissionID := uuid.New().String()
	filename := filepath.Join(s.opts.UploadDir, submissionID+".ipynb")
	err = ioutil.WriteFile(filename, b, 0700)
	glog.V(3).Infof("Uploaded %d bytes", len(b))
	if err != nil {
		return fmt.Errorf("error writing uploaded file: %s", err)
	}
	// Store submission ID inside the metadata.
	data := make(map[string]interface{})
	err = json.Unmarshal(b, &data)
	if err != nil {
		return fmt.Errorf("could not parse submission as JSON: %s", err)
	}
	var metadata map[string]interface{}
	v, ok := data["metadata"]
	if ok {
		metadata, ok = v.(map[string]interface{})
	}
	if !ok {
		metadata = make(map[string]interface{})
		data["metadata"] = metadata
	}
	metadata["submission_id"] = submissionID
	b, err = json.Marshal(data)
	if err != nil {
		return err
	}
	glog.V(3).Infof("Checking %d bytes", len(b))
	err = s.scheduleCheck(b)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/plain")
	if s.opts.AllowCORSOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", s.opts.AllowCORSOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	glog.V(5).Infof("Uploaded: %s", string(b))
	fmt.Fprintf(w, "/report/"+submissionID)
	return nil
}

func (s *Server) scheduleCheck(content []byte) error {
	return s.opts.Channel.Post(s.opts.QueueName, content)
}

func (s *Server) ListenForReports(ch <-chan []byte) {
	for b := range ch {
		glog.V(3).Infof("Received %d byte report", len(b))
		glog.V(5).Infof("Received: %s", string(b))
		data := make(map[string]interface{})
		err := json.Unmarshal(b, &data)
		if err != nil {
			glog.Errorf("data: %q, error: %s", string(b), err)
			continue
		}
		v, ok := data["submission_id"]
		if !ok {
			glog.Errorf("Report did not have submission_id: %#v", data)
			continue
		}
		submissionID, ok := v.(string)
		if !ok {
			glog.Errorf("submission_id was not a string, but %s",
				reflect.TypeOf(v))
			continue
		}
		// TODO(salikh): Write a pretty report instead.
		filename := filepath.Join(s.opts.UploadDir, submissionID+".txt")
		err = ioutil.WriteFile(filename, b, 0775)
		if err != nil {
			glog.Errorf("Error writing to %q: %s", filename, err)
			continue
		}
	}
}

// uploadForm provides a simple web form for manual uploads.
func (s *Server) uploadForm(w http.ResponseWriter, req *http.Request) error {
	if s.opts.UseOpenID {
		err := s.authenticate(w, req)
		if err != nil {
			return err
		}
	}
	if req.Method != "GET" {
		return fmt.Errorf("Unsupported method %s on %s", req.Method, req.URL.Path)
	}
	glog.Infof("GET %s", req.URL.Path)
	_, err := w.Write([]byte(uploadHTML))
	return err
}

const uploadHTML = `<!DOCTYPE html>
<title>Upload form</title>
<form method="POST" action="/upload" enctype="multipart/form-data">
	<input type="file" name="notebook">
	<input type="submit" value="Upload">
</form>`

const favIconBase64 = `
AAABAAEAICAAAAEAIACoEAAAFgAAACgAAAAgAAAAQAAAAAEAIAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD/BAAA/18AAP+2AAD/5wAA//oAAP/0
AAD/1wAA/5gAAP8sAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA/xcAAP/KAAD//wAA
//8AAP//AAD//wAA//8AAP//AAD//wAA//0AAP90AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAP8B
AAD/wAAA//8AAP/4AAD/ewAA/yMAAP8FAAD/DAAA/0AAAP+8AAD//wAA//8AAP9SAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAA/0cAAP//AAD//wAA/1UAAAAAAAAAAAAAAAAAAAAAAAAAAAAA/wMAAP/BAAD/
/wAA/9gAAP8BAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD/mgAA//8AAP/aAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAA/0gAAP//AAD//wAA/ywAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAP/MAAD//wAA/50AAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD/DAAA//4AAP//AAD/XgAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA/+AA
AP//AAD/gQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD/7wAA//8AAP9yAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAD/6AAA//8AAP94AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAP/k
AAD//wAA/3sAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAP/oAAD//wAA/3gAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAA/+QAAP//AAD/fAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA/+gAAP//AAD/eAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD/5AAA//8AAP98AAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD/6AAA
//8AAP94AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAP/kAAD//wAA/3wAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAP/oAAD//wAA/3gAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA/+QA
AP//AAD/fAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA/+gAAP//AAD/eAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAD/5AAA//8AAP98AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD/6AAA//8AAP94AAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAP/kAAD//wAA/3wAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAP/oAAD/
/wAA/3gAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA/+QAAP//AAD/fAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAA/+gAAP//AAD/eAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD/5AAA
//8AAP98AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD/6AAA//8AAP94AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAP/kAAD//wAA/3wAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAP/oAAD//wAA/3gAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA/+QAAP//AAD/fAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA////
///////////////////////////////////gD///wAf//4AD//+Hwf//j+H//4/h//+P8f//j/H/
/4/x//+P8f//j/H//4/x//+P8f//j/H//4/x//+P8f//j/H//4/x////////////////////////
//////////////8=
`

var favIcon []byte

func init() {
	var err error
	favIcon, err = base64.StdEncoding.DecodeString(favIconBase64)
	if err != nil {
		panic(err)
	}
}
