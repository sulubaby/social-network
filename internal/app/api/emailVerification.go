package api

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"
	"net/smtp"
	"os"
	"regexp"
	"social/database/users"
	"social/internal/helpers"
	"social/internal/validation"
	"strings"
	"time"
)

var sixDigitCode = regexp.MustCompile(`^[0-9]{6}$`)

/*
function used to generate OTP for email verification

Parameters:
	string -> email
	error -> nil if success
*/
func generateEmailCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}

/*
triesLabel returns a formatted string describing the number of verification
attempts, using the singular form for one attempt and the plural form for
multiple attempts.

Parameters:
	n int
		-> number of attempt
	
Returns:
	string
		-> formatted text

*/
func triesLabel(n int) string {
	if n == 1 {
		return "1 try"
	}

	return fmt.Sprintf("%d tries", n)
}
/*
SendEmailCode handles requests to generate and send an email verification code.
It validates the requested email, generates a verification code, saves it in
the database, enforces the code request cooldown, and sends the code to the
provided email address. It also returns an error response if the email is
already registered, the request is made during the cooldown period, or the
code cannot be generated, saved, or sent.
*/
func (app *App) SendEmailCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid request",
		})
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))

	if err := validation.ValidateEmail(email); err != nil {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	code, err := generateEmailCode()
	if err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not generate verification code",
		})
		return
	}

	retryAfter, err := users.SaveEmailCode(app.DB, email, code, time.Now())
	if err != nil {
		switch {
		case errors.Is(err, users.ErrEmailTaken):
			helpers.WriteJson(w, http.StatusConflict, map[string]any{
				"status":  false,
				"message": "This email is already registered.",
			})
		case errors.Is(err, users.ErrCodeCooldown):
			seconds := int(math.Ceil(retryAfter.Seconds()))
			helpers.WriteJson(w, http.StatusTooManyRequests, map[string]any{
				"status":     false,
				"message":    fmt.Sprintf("A code was already sent. You can request a new one in %d seconds.", seconds),
				"retryAfter": seconds,
			})
		default:
			log.Println(err)
			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "could not save verification code",
			})
		}
		return
	}

	if err := app.SendVerificationEmail(email, code); err != nil {
		log.Println(err)

		if delErr := users.DeleteEmailCode(app.DB, email); delErr != nil {
			log.Println(delErr)
		}

		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not send verification email",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":   true,
		"message":  "verification code sent",
		"cooldown": int(users.EmailCodeCooldown.Seconds()),
		"attempts": users.MaxEmailCodeAttempts,
	})
}

/*
VerifyEmail handles requests to verify an email address using the provided
6-digit verification code. It validates the email and code, checks the code
against the stored verification record, and returns a verification token
when the code is correct. It also handles incorrect, expired, missing, and
excessive verification attempts and returns the remaining attempts when
applicable.
*/
func (app *App) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid request",
		})
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	code := strings.TrimSpace(req.Code)

	if email == "" {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "email is required",
		})
		return
	}

	if !sixDigitCode.MatchString(code) {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "Enter the 6-digit code.",
		})
		return
	}

	token, attemptsLeft, err := users.VerifyEmailCode(app.DB, email, code, time.Now())
	if err == nil {
		helpers.WriteJson(w, http.StatusOK, map[string]any{
			"status":   true,
			"verified": true,
			"token":    token,
			"message":  "email verified successfully",
		})
		return
	}

	switch {
	case errors.Is(err, users.ErrCodeMismatch):
		message := "Incorrect code. No tries left, request a new code."
		if attemptsLeft > 0 {
			message = fmt.Sprintf("Incorrect code. %s left.", triesLabel(attemptsLeft))
		}
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":       false,
			"message":      message,
			"attemptsLeft": attemptsLeft,
		})
	case errors.Is(err, users.ErrTooManyAttempts):
		helpers.WriteJson(w, http.StatusTooManyRequests, map[string]any{
			"status":       false,
			"message":      "Too many incorrect attempts. Request a new code.",
			"attemptsLeft": 0,
		})
	case errors.Is(err, users.ErrCodeExpired):
		helpers.WriteJson(w, http.StatusGone, map[string]any{
			"status":       false,
			"message":      "This code has expired. Request a new code.",
			"attemptsLeft": 0,
		})
	case errors.Is(err, users.ErrCodeNotFound):
		helpers.WriteJson(w, http.StatusNotFound, map[string]any{
			"status":       false,
			"message":      "No active code for this email. Request a new code.",
			"attemptsLeft": 0,
		})
	default:
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not verify email",
		})
	}
}

/*
function used to send email verification code to a specefic email.
It use SMT Protocol to send the email by providing a username (email). passoword (please do not steal it)
and a host which in this case gmail.

Parameters:
	to -> email address
	code -> verification code

Returns:
	error -> nil if success
*/
func (app *App) SendVerificationEmail(to string, code string) error {
	// local testing: print the code in the server log instead of sending a real email
	if os.Getenv("ORBIT_EMAIL_DEV_LOG") == "1" {
		log.Printf("verification code for %s: %s", to, code)
		return nil
	}

	auth := smtp.PlainAuth(
		"",
		app.EmailAddress,
		app.EmailPassword,
		"smtp.gmail.com",
	)

	message := []byte(
		"From: " + app.EmailAddress + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: Email Verification Code\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"\r\n" +
			"<html>" +
			"<body>" +
			"<h2>Email Verification</h2>" +
			"<p>Your verification code is:</p>" +
			"<h1>" + code + "</h1>" +
			"<p>This code expires in 10 minutes.</p>" +
			"</body>" +
			"</html>",
	)

	return smtp.SendMail(
		"smtp.gmail.com:587",
		auth,
		app.EmailAddress,
		[]string{to},
		message,
	)
}
