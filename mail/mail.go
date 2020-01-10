package mail

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"text/template"
	"time"

	gomail "gopkg.in/gomail.v2"
)

var reportTmpl = template.Must(template.New("report").Parse(`Error: {{ .Err }}

Traceback:

{{ printf "%s" .Stack }}

Request:

{{ .RequestInfo }}

URI: {{ .Request.RequestURI }}
IP: {{ .Request.RemoteAddr }}
`))

// ErrNotImplemented used when the functionality is not implemented yet.
var ErrNotImplemented = errors.New("not implemented")

// DefaultConfig is a default config for a server in a cloud.
var DefaultConfig = Config{
	Host:     "localhost",
	Port:     25,
	Insecure: true,
}

// Config contains basic information needed for mail sending.
type Config struct {
	Host, User, Password string
	Port                 int
	Insecure             bool
}

// Message represents a message to be sent.
type Message struct {
	From, Subject, Body, BodyHTML string
	To                            []string
	Created                       time.Time
}

// SendErrorInsecure sends an error report to specified receivers insecurely.
// It means that certificates must not be trusted.
func SendErrorInsecure(config Config, reportErr error, r *http.Request, code int, from, subj string, to []string) error {
	var requestInfo string
	if r != nil {
		requestInfo = fmt.Sprintf("%s %s %d %s referer: %s remote_addr: %v | %v",
			r.Method, r.URL.String(), code, r.UserAgent(), r.Referer(), r.RemoteAddr, r.Header.Get("X-Real-IP"))
	}

	var body bytes.Buffer
	report := struct {
		Err         error
		Stack       []byte
		RequestInfo string
		Request     *http.Request
	}{
		Err:         reportErr,
		Stack:       debug.Stack(),
		RequestInfo: requestInfo,
		Request:     r,
	}

	if err := reportTmpl.Execute(&body, report); err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to...)
	m.SetHeader("Subject", subj)
	m.SetBody("text/plain", body.String())
	d := gomail.NewDialer(config.Host, config.Port, config.User, config.Password)
	if config.Insecure {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	} else {
		return ErrNotImplemented
	}
	log.Println("sending a report about an error")
	return d.DialAndSend(m)
}

// Send sends a message to recipients.
func Send(cfg Config, msg Message) error {
	m := gomail.NewMessage()
	m.SetHeader("From", msg.From)
	m.SetHeader("To", msg.To...)
	m.SetHeader("Subject", msg.Subject)
	if len(msg.BodyHTML) > 0 {
		m.SetBody("text/html", msg.BodyHTML)
	} else {
		m.SetBody("text/plain", msg.Body)
	}
	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Password)
	if cfg.Insecure {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	} else {
		return ErrNotImplemented
	}
	return d.DialAndSend(m)
}
