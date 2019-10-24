// Package uploadserver provides an implemenation of the upload server
// that accepts student notebook uploads (submissions), posts them to the
// message queue for grading, listens for the reports on the message queue
// and makes reports available on the web.
package uploadserver

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"time"

	"cloud.google.com/go/storage"
	"github.com/golang/glog"
	"github.com/google/prog-edu-assistant/autograder"
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
	// AllowCORS specifies whether cross-origin requests are allowed.
	AllowCORS bool
	// GradeLocally is boolean, if true specifies whether the autograding task
	// should be invoked locally without using a message queue.
	GradeLocally bool
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
	// SecureCookie specifies whether the cookie must have Secure attribute or not.
	SecureCookie bool
	// HashSalt should be set to a random string. It is used for hashing student
	// ids.
	HashSalt string
	// StaticDir is set to the path of the directory exposed at /static URL.
	StaticDir string
	// HTTPRedirectPort controls the HTTP redirect to HTTPS.
	HTTPRedirectPort int
	// Autograder contains the setup for the local grading environment,
	// only used when GradeLocally is true.
	*autograder.Autograder
	// LogToBucket specifies whether the server should write
	// logs to the Google Cloud Storage bucket.
	LogToBucket bool
	// LogBucketName specifies the bucket name. This is only
	// used if LogToBucket is true.
	LogBucketName string
	// ProjectID is the GCP project ID that is used for Google
	// Cloud Storage access if LogToBucket is true.
	ProjectID string
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
	s.cookieStore.Options = &sessions.Options{
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   s.opts.SecureCookie,
	}
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
	if s.opts.StaticDir != "" {
		glog.Infof("Registering static file server on /static from %s", opts.StaticDir)
		mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(opts.StaticDir))))
	}
	mux.Handle("/", handleError(s.uploadForm))
	return s
}

const UserSessionName = "user_session"

// hashId uses cryptographic hash (sha224) and a secret salt
// to hash the user id (email address) into a hash.
func (s *Server) hashId(id string) string {
	b := sha256.Sum224([]byte(s.opts.HashSalt + id))
	return hex.EncodeToString(b[:])
}

// ListenAndServe starts the server similarly to http.ListenAndServe.
func (s *Server) ListenAndServe(addr string) error {
	err := os.MkdirAll(s.opts.UploadDir, 0700)
	if err != nil {
		return fmt.Errorf("error creating upload dir %q: %s", s.opts.UploadDir, err)
	}
	return http.ListenAndServe(addr, s.mux)
}

// ListenAndServeTLS starts a server using HTTPS.
func (s *Server) ListenAndServeTLS(addr, certFile, keyFile string) error {
	if s.opts.HTTPRedirectPort != 0 {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			p := req.URL.Path
			if p != "" && p[0] != '/' {
				p = "/" + p
			}
			http.Redirect(w, req, s.opts.ServerURL+p, http.StatusTemporaryRedirect)
		})
		go http.ListenAndServe(fmt.Sprintf(":%d", s.opts.HTTPRedirectPort), mux)
	}
	config := &tls.Config{MinVersion: tls.VersionTLS10}
	httpserver := &http.Server{
		Addr:      addr,
		TLSConfig: config,
		Handler:   s.mux,
	}
	return httpserver.ListenAndServeTLS(certFile, keyFile)
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
				if status == http.StatusUnauthorized {
					// Provide a convenience login link.
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.WriteHeader(int(status))
					fmt.Fprintf(w, `<html>
<title>Not logged in</title>
<h3>Not logged in</h3>
Click here to log in: <a href='/login'>Log in</a>.`)
					return
				}
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
// generic error and stops autoreloading. Note that if the user manually
// refreshes the page later, the same autoreload process is repeated. This
// process is designed to handle the case of workers being overloaded with
// grading work and producing reports with long delay.
func (s *Server) handleReport(w http.ResponseWriter, req *http.Request) error {
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
		if reloadMs > 20000 {
			// Reset for retry.
			reloadMs = 500
			s.reportTimestamp[basename] = time.Now()
		}
		if reloadMs > 10000 {
			fmt.Fprintf(w, `<title>Something weng wrong</title>
<h2>Error</h2>
Something went wrong, please reload this page.
If reloading does not help, wait a minute and retry your upload.
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
	return s.renderReport(w, basename, b)
}

func (s *Server) renderReport(w http.ResponseWriter, submissionID string, reportData []byte) error {
	data := make(map[string]interface{})
	err := json.Unmarshal(reportData, &data)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var exerciseIDs []string
	var reports = make(map[string]string)
	// Extract reports
	fill := &reportFill{
		Title: "Report for " + submissionID,
	}
	for exerciseID, x := range data {
		m, ok := x.(map[string]interface{})
		if !ok {
			continue
		}
		if report, ok := m["report"]; ok {
			html, ok := report.(string)
			if !ok {
				return fmt.Errorf("expected report to be a string, got %s", reflect.TypeOf(report))
			}
			reports[exerciseID] = html
			exerciseIDs = append(exerciseIDs, exerciseID)
		}
	}
	// Sort reports by exercise_id.
	sort.Strings(exerciseIDs)
	// Just concatenate all reports in order.
	for _, exerciseID := range exerciseIDs {
		fill.Exercises = append(fill.Exercises, exerciseFill{exerciseID, template.HTML(reports[exerciseID])})
	}
	if len(fill.Exercises) == 0 {
		fill.ErrorMessage = fmt.Sprintf("Report %s contained no checks", submissionID)
	}
	return reportTmpl.Execute(w, fill)
}

type exerciseFill struct {
	ReportTitle string
	ReportHTML  template.HTML
}

type reportFill struct {
	Title        string
	Exercises    []exerciseFill
	ErrorMessage string
}

var reportTmpl = template.Must(template.New("reportTmpl").Parse(`
<title>{{.Title}}</title>
<style type='text/css'>
h2 {
  color: #697;
  font-size: 10pt;
  font-family: Verdana, Arial, sans-serif;
  margin-top: 2em;
}
.message {
  font-size: 14pt;
  font-weight: medium;
}
.ico {
  font-size: 16pt;
  font-weight: bold;
  padding: 0px 2px 0px 2px;
  margin: 10px 4px 1px 4px;
  background: #EEE;
  border: 1pt solid #DDD;
  border-radius: 3pt;
}
.green {
  color: #2F2;
}
.red {
  color: #F22;
}
.code {
  white-space: pre;
  font-family: monospace;
  background: #F0F0F0;
  padding: 3pt;
  margin: 4pt;
  border: 1pt solid #DDD;
  border-radius: 3pt;
}
.code ol {
  margin: 0px;
  padding: 0px;
  padding-inline-start: 22pt;
  margin-block-start: 0em;
  margin-block-end: 0em;
  line-height: 10%;
}
.code ol li {
  margin: 0px;
  padding: 0px;
  line-height: 120%;
}
.code ol li:nth-child(odd) {
  background: #F8F8F8;
}
.code li:last-child {
  margin-bottom: 0px;
}

/*
 * Based on default theme
 * from http://github.com/google/code-prettify.
 */

/* SPAN elements with the classes below are added by prettyprint. */
.pln { color: #000 }  /* plain text */

@media screen {
  .str { color: #080 }  /* string content */
  .kwd { color: #008 }  /* a keyword */
  .com { color: #800 }  /* a comment */
  .typ { color: #606 }  /* a type name */
  .lit { color: #066 }  /* a literal value */
  /* punctuation, lisp open bracket, lisp close bracket */
  .pun, .opn, .clo { color: #660 }
  .tag { color: #008 }  /* a markup tag name */
  .atn { color: #606 }  /* a markup attribute name */
  .atv { color: #080 }  /* a markup attribute value */
  .dec, .var { color: #606 }  /* a declaration; a variable name */
  .fun { color: red }  /* a function name */
}

/* Use higher contrast and text-weight for printable form. */
@media print, projection {
  .str { color: #060 }
  .kwd { color: #006; font-weight: bold }
  .com { color: #600; font-style: italic }
  .typ { color: #404; font-weight: bold }
  .lit { color: #044 }
  .pun, .opn, .clo { color: #440 }
  .tag { color: #006; font-weight: bold }
  .atn { color: #404 }
  .atv { color: #060 }
}

/* Put a border around prettyprinted code snippets. */
pre.prettyprint { padding: 2px; border: 1px solid #888 }

/* Specify class=linenums on a pre to get line numbering */
ol.linenums { margin-top: 0; margin-bottom: 0 } /* IE indents via margin-left */
li.L0,
li.L1,
li.L2,
li.L3,
li.L5,
li.L6,
li.L7,
li.L8 { list-style-type: none }
/* Alternate shading for lines */
li.L1,
li.L3,
li.L5,
li.L7,
li.L9 { background: #eee }
</style>
{{range .Exercises}}
{{if .ReportTitle}}
<h2>{{.ReportTitle}}</h2>
{{end}}
{{.ReportHTML}}
{{end}}
{{if .ErrorMessage}}
<div class='error'>
{{.ErrorMessage}}
</div>
{{end}}
`))

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
		session.Options.MaxAge = -1
		delete(session.Values, "hash")
		session.Save(req, w)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(fmt.Sprintf("<title>Forbidden</title>User %s is not authorized.<br>"+
			"Try a different Google account. <a href='https://mail.google.com/mail/logout'>Log out of Google</a>.", profile.Email)))
		return nil
	}
	// Restrict the cookie by 1h, HttpOnly and Secure (if configured).
	session.Options = &sessions.Options{
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   s.opts.SecureCookie,
	}
	// Instead of email, we store a salted cryptographic hash (pseudonymous id).
	session.Values["hash"] = s.hashId(profile.Email)
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
	hash, ok := session.Values["hash"]
	if ok {
		fmt.Fprintf(w, "Logged in as %s. <a href='/logout'>Log out link</a>.", hash)
		fmt.Fprintf(w, "<p>Click here to <a href='/'>upload</a> a notebook manually, or "+
			"<strong>You can close this window and retry upload from the Jupyter notebook.</strong>")
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
	session.Options.MaxAge = -1
	delete(session.Values, "hash")
	session.Save(req, w)
	http.Redirect(w, req, "/profile", http.StatusTemporaryRedirect)
	return nil
}

// authenticate handles the authentication. If authentication or authorization
// was not successful, it returns an error. Normally it returns the user hash.
func (s *Server) authenticate(w http.ResponseWriter, req *http.Request) (string, error) {
	session, err := s.cookieStore.Get(req, UserSessionName)
	if err != nil {
		session.Options.MaxAge = -1
		return "", fmt.Errorf("cookieStore.Get returned error %s", err)
	}
	hash, ok := session.Values["hash"].(string)
	glog.V(3).Infof("authenticate %s: hash=%s", req.URL, session.Values["hash"])
	if !ok || hash == "" {
		return "", httpError(http.StatusUnauthorized)
	}
	return hash, nil
}

const maxUploadSize = 1048576

// handleUpload handles the upload requests via web form.
func (s *Server) handleUpload(w http.ResponseWriter, req *http.Request) error {
	glog.Infof("%s %s", req.Method, req.URL.Path)
	if s.opts.AllowCORS {
		origin := "*"
		if len(req.Header["Origin"]) > 0 {
			origin = req.Header["Origin"][0]
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "1800")
		// X-Report-Url header is used to report back the link to the report.
		w.Header().Set("Access-Control-Expose-Headers", "X-Report-Url")
		if req.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST")
		}
	}
	if req.Method == "OPTIONS" {
		return nil
	}
	userHash := "unknown"
	if s.opts.UseOpenID {
		var err error
		userHash, err = s.authenticate(w, req)
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
	submissionFilename := filepath.Join(s.opts.UploadDir, submissionID+".ipynb")
	err = ioutil.WriteFile(submissionFilename, b, 0700)
	glog.V(3).Infof("Uploaded %d bytes", len(b))
	if err != nil {
		return fmt.Errorf("error writing uploaded file: %s", err)
	}
	// Store user hash and submission ID inside the metadata.
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
	metadata["user_hash"] = userHash
	metadata["timestamp"] = time.Now().Unix()
	b, err = json.Marshal(data)
	if err != nil {
		return err
	}
	// submissionID is an UUID, so it does not require escaping.
	reportURL := "/report/" + submissionID
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// X-Report-Url header is used to report back the link to the report.
	w.Header().Set("X-Report-Url", reportURL)
	glog.V(5).Infof("Uploaded: %s", string(b))
	if s.opts.LogToBucket && s.opts.LogBucketName != "" {
		ctx := req.Context()
		client, err := storage.NewClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to create Cloud Storage client: %w", err)
		}
		bucket := client.Bucket(s.opts.LogBucketName)
		logW := bucket.Object(submissionID + ".ipynb").NewWriter(ctx)
		n, err := logW.Write(b)
		if err != nil {
			return fmt.Errorf("error writing log to bucket %q: %w",
				s.opts.LogBucketName, err)
		}
		err = logW.Close()
		if err != nil {
			return fmt.Errorf("error closing log writer: %w", err)
		}
		glog.V(5).Infof("Written %d bytes to %s to log bucket %s", n, submissionID+".ipynb", s.opts.LogBucketName)
	}
	if !s.opts.GradeLocally {
		glog.V(3).Infof("Checking %d bytes", len(b))
		err = s.scheduleCheck(b)
		if err != nil {
			return err
		}
		err = uploadResultTmpl.Execute(w, reportURL)
		if err != nil {
			glog.Error(err)
		}
		return nil
	}
	// Grade locally
	report, err := s.opts.Autograder.Grade(b)
	if err != nil {
		return fmt.Errorf("error grading: %s", err)
	}
	reportFilename := filepath.Join(s.opts.UploadDir, submissionID+".txt")
	err = ioutil.WriteFile(reportFilename, report, 0775)
	if err != nil {
		return fmt.Errorf("error writing to %q: %s", reportFilename, err)
	}
	if s.opts.LogToBucket && s.opts.LogBucketName != "" {
		ctx := req.Context()
		client, err := storage.NewClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to create Cloud Storage client: %w", err)
		}
		bucket := client.Bucket(s.opts.LogBucketName)
		logW := bucket.Object(submissionID + ".txt").NewWriter(ctx)
		n, err := logW.Write(report)
		if err != nil {
			return fmt.Errorf("error writing log to bucket %q: %w",
				s.opts.LogBucketName, err)
		}
		err = logW.Close()
		if err != nil {
			return fmt.Errorf("error closing log writer: %w", err)
		}
		glog.V(5).Infof("Written %d bytes to %s to log bucket %s", n, submissionID+".txt", s.opts.LogBucketName)
	}
	return s.renderReport(w, submissionID, report)
}

var uploadResultTmpl = template.Must(template.New("uploadResultTmpl").Parse(`
<html>
<title>Upload completed</title>
<link rel='stylesheet' type='text/css' href='/static/style.css'/>
<h2>Upload succeeded</h2>
Click here for the <a href='{{.}}'>Report</a>.
`))

func (s *Server) scheduleCheck(content []byte) error {
	return s.opts.Channel.Post(s.opts.QueueName, content)
}

func (s *Server) ListenForReports(ch <-chan []byte) {
	glog.Infof("Listening for reports")
	for b := range ch {
		glog.V(1).Infof("Received %d byte report", len(b))
		glog.V(5).Infof("Received report:\n%s\n--\n", string(b))
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
		_, err := s.authenticate(w, req)
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
<title>Upload notebook | Prog-edu-assistant</title>
<link rel='stylesheet' type='text/css' href='/static/style.css'/>
<h2>Notebook upload</h2>
You can upload a notebook for checking.
Only notebooks from https://github.com/salikh/student-notebooks are accepted for checking.
<p>
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
