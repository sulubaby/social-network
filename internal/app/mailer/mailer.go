package mailer

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html/template"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

const (
	smtpHost = "smtp.gmail.com"
	smtpPort = "587"
)

/*
Config holds the settings the mailer needs to send emails: the address the emails are sent from
(EmailAddress) and the password used to log in to it (EmailPassword). in this project the values come
from the EMAIL_ADDRESS and EMAIL_PASSWORD environment variables (see envOr in the routes package).
*/
type Config struct {
	EmailPassword string
	EmailAddress  string
}

/*
Mailer is used to send emails (like the verification code email). it keeps its Config private, so it can
only be created with New.
*/
type Mailer struct {
	cfg Config
}

/*
otpView is the data given to the OTP email template (otpTemplate). it is only used to fill in the html of
the verification code email:
	-> Digits: the digits of the code as separate strings, so the template can show each one in its own box
	-> Code: the full code as one string
	-> Minutes: how many minutes the code stays valid
	-> Tries: how many wrong attempts are allowed before the code is locked
*/
type otpView struct {
	Digits  []string
	Code    string
	Minutes int
	Tries   int
}

/*
otpTemplate is the parsed html template of the verification code email (from otpHTML). it is parsed once
when the program starts. template.Must makes the program panic at startup if the template is invalid, so a
broken template is found immediately and not when the first email is sent.
*/
var otpTemplate = template.Must(template.New("otp").Parse(otpHTML))

/*
function used to create a new Mailer from a Config. it removes all the spaces from the email password
before saving it, because Gmail app passwords are shown in groups separated by spaces (like
"abcd efgh ijkl mnop") but must be used without them. so the password can be copied as it is shown.

it takes the Config by value, so the config of the caller is not changed.
*/
func New(cfg Config) *Mailer {
	cfg.EmailPassword = strings.ReplaceAll(cfg.EmailPassword, " ", "")

	return &Mailer{cfg: cfg}
}

/*
function used to encode data as base64 and split it into lines of 76 characters, separated by "\r\n".
emails (MIME) do not allow lines longer than 76 characters in a base64 body, so the encoded text must be
wrapped. the last line is shorter (or exactly 76) and has no "\r\n" after it, so the caller adds the line
break itself.
*/
func wrap76(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)

	var b strings.Builder

	for len(encoded) > 76 {
		b.WriteString(encoded[:76])
		b.WriteString("\r\n")
		encoded = encoded[76:]
	}

	b.WriteString(encoded)

	return b.String()
}

/*
function used to generate a random id (24 hex characters, from 12 random bytes). it is used to make the
boundary of the email and the Message-ID header, so every email gets a different one.
*/
func randomID() string {
	b := make([]byte, 12)
	rand.Read(b)

	return hex.EncodeToString(b)
}

/*
function used to build the raw email message (headers and body) that is sent over SMTP. it takes the
receiver, the subject, a plain text version and an html version of the body, and returns the full message
as bytes.

the message is "multipart/alternative", so the email app shows the html version and falls back to the plain
text one if it can not show html. both parts are base64 encoded and wrapped at 76 characters (see wrap76).

the headers it sets:
	-> From: the display name "social network" with the address from the config
	-> To, Subject, Date (current time, RFC 1123Z format)
	-> Message-ID: a random id with the domain of the sender address (if the address has no "@", it uses
	   "socialnetwork.app")
	-> MIME-Version and Content-Type (with a random boundary that separates the two parts)

this function does not send anything, it only builds the message. the lines end with "\r\n" as SMTP
requires.
*/
func (m *Mailer) build(to, subject, plain, html string) []byte {
	boundary := "social-" + randomID()
	from := mail.Address{Name: "social network", Address: m.cfg.EmailAddress}

	domain := "socialnetwork.app"

	if at := strings.LastIndex(m.cfg.EmailAddress, "@"); at >= 0 {
		domain = m.cfg.EmailAddress[at+1:]
	}

	var b bytes.Buffer

	fmt.Fprintf(&b, "From: %s\r\n", from.String())
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: <%s@%s>\r\n", randomID(), domain)
	b.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary)

	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	b.WriteString(wrap76([]byte(plain)))
	b.WriteString("\r\n")

	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	b.WriteString(wrap76([]byte(html)))
	b.WriteString("\r\n")

	fmt.Fprintf(&b, "--%s--\r\n", boundary)

	return b.Bytes()
}

/*
function used to send a ready message (made by build) to an email address through the SMTP server
(smtpHost / smtpPort). it opens a new connection for every email, there is no connection reuse.

the steps, in order:
	-> connect over tcp (10 seconds timeout for connecting)
	-> set a deadline of 30 seconds for the whole conversation with the server, so a stuck server can not
	   block it forever
	-> upgrade the connection to TLS (STARTTLS) BEFORE logging in, so the password is never sent in clear
	-> log in with the address and password from the config (PLAIN auth)
	-> tell the server who the mail is from (MAIL) and who it is for (RCPT)
	-> write the message (DATA) and close the writer, which is what actually delivers it
	-> say goodbye to the server (QUIT)

if any step fails it stops and returns that error. the connection is always closed when the function ends.
this function is synchronous, so the caller waits until the email is sent or fails.

returns:
	-> nil if the server accepted the email
	-> error if anything failed
*/
func (m *Mailer) send(to string, msg []byte) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(smtpHost, smtpPort), 10*time.Second)
	if err != nil {
		return err
	}

	conn.SetDeadline(time.Now().Add(30 * time.Second))

	client, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		conn.Close()
		return err
	}
	defer client.Close()

	if err := client.StartTLS(&tls.Config{ServerName: smtpHost}); err != nil {
		return err
	}

	auth := smtp.PlainAuth("", m.cfg.EmailAddress, m.cfg.EmailPassword, smtpHost)

	if err := client.Auth(auth); err != nil {
		return err
	}

	if err := client.Mail(m.cfg.EmailAddress); err != nil {
		return err
	}

	if err := client.Rcpt(to); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}

	if _, err := w.Write(msg); err != nil {
		return err
	}

	if err := w.Close(); err != nil {
		return err
	}

	return client.Quit()
}

/*
function used to send the verification code email (the code made by Store.Issue) to an email address. it
fills the html template (otpTemplate) with the code split into digits, the number of minutes the code is
valid (the ttl, rounded down to whole minutes) and the number of tries allowed. it also makes a plain text
version with the same information, builds the message and sends it.

the subject of the email contains the code, so the user can see it without opening the email.

returns:
	-> nil if the email was sent
	-> error if the template failed or the email could not be sent (see send)
*/
func (m *Mailer) SendVerificationCode(to, code string, ttl time.Duration, tries int) error {
	view := otpView{
		Digits:  strings.Split(code, ""),
		Code:    code,
		Minutes: int(ttl.Minutes()),
		Tries:   tries,
	}

	var htmlBody bytes.Buffer

	if err := otpTemplate.Execute(&htmlBody, view); err != nil {
		return err
	}

	plain := fmt.Sprintf(
		"social network - VERIFICATION\r\n\r\nYour verification code is: %s\r\n\r\nIt expires in %d minutes and you have %d tries.\r\n\r\nDid not request this? Ignore this email.\r\n",
		code, view.Minutes, tries,
	)

	msg := m.build(to, "social network - Your verification code is "+code, plain, htmlBody.String())

	return m.send(to, msg)
}
