package otp

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
)

const (
	CodeLength      = 6
	MaxAttempts     = 3
	CodeTTL         = 10 * time.Minute
	Cooldown        = 60 * time.Second
	TokenTTL        = 15 * time.Minute
	MaxSendsPerHour = 5
)

var (
	ErrNoCode    = errors.New("no code")
	ErrExpired   = errors.New("code expired")
	ErrLocked    = errors.New("too many attempts")
	ErrWrongCode = errors.New("wrong code")
)

type RateLimitError struct {
	RetryAfter int
	Message    string
}

func (e *RateLimitError) Error() string {
	return e.Message
}

type entry struct {
	hash         [32]byte
	expires      time.Time
	attemptsLeft int
	lastSent     time.Time
	sends        []time.Time
}

type token struct {
	email   string
	expires time.Time
}

type Store struct {
	mu     sync.Mutex
	codes  map[string]*entry
	tokens map[string]token
}

func NewStore() *Store {
	return &Store{
		codes:  make(map[string]*entry),
		tokens: make(map[string]token),
	}
}

func hashCode(email, code string) [32]byte {
	return sha256.Sum256([]byte(email + ":" + code))
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

/*
function used to clean the expired data out of the store so it does not grow forever. it takes the current
time as a parameter (instead of calling time.Now itself) so the time can be controlled when testing.

it goes over two maps:
	-> codes: the email verification codes. a code is deleted only when it has expired AND the last time
	   a code was sent to that email was more than an hour ago, so the resend limit of an email is not
	   forgotten while it still applies.
	-> tokens: the verification tokens. a token is deleted as soon as it has expired.

this function does not lock the store, so the caller must already hold the lock when calling it.
*/
func (s *Store) purge(now time.Time) {
	for email, e := range s.codes {
		if now.After(e.expires) && now.Sub(e.lastSent) > time.Hour {
			delete(s.codes, email)
		}
	}

	for value, t := range s.tokens {
		if now.After(t.expires) {
			delete(s.tokens, value)
		}
	}
}

/*
function used to issue a new 6 digit verification code for an email. it locks the store, purges the
expired data, then checks the rate limits of that email before creating the code. if the email is
allowed, it generates a random code, saves only its hash (never the code itself) and returns the code so
the caller can send it by mail.

the rate limits, checked in this order:
	-> cooldown: if the last code was sent less than Cooldown ago, it fails with a RateLimitError that
	   tells how many seconds are left to wait.
	-> hourly limit: the sends older than one hour are dropped from the list, then if the email still has
	   MaxSendsPerHour or more sends in the last hour, it fails with a RateLimitError that tells how long
	   until the oldest send leaves the one hour window.

if the email has no entry yet, a new one is created. when a code is issued, the entry is reset: the hash
is replaced (so the previous code stops working), the expiry is set to CodeTTL from now, the attempts are
set back to MaxAttempts, and the send time is added to the list of sends.

returns:
	-> the code, nil on success
	-> "", *RateLimitError if the email must wait
	-> "", error if the random number could not be generated
*/
func (s *Store) Issue(email string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	s.purge(now)

	e := s.codes[email]

	if e != nil {
		if wait := Cooldown - now.Sub(e.lastSent); wait > 0 {
			seconds := int(wait.Seconds()) + 1

			return "", &RateLimitError{
				RetryAfter: seconds,
				Message:    fmt.Sprintf("Please wait %ds before requesting another code", seconds),
			}
		}

		recent := e.sends[:0]

		for _, sentAt := range e.sends {
			if now.Sub(sentAt) < time.Hour {
				recent = append(recent, sentAt)
			}
		}

		e.sends = recent

		if len(e.sends) >= MaxSendsPerHour {
			wait := time.Hour - now.Sub(e.sends[0])
			seconds := int(wait.Seconds()) + 1

			return "", &RateLimitError{
				RetryAfter: seconds,
				Message:    "Too many codes requested. Try again later",
			}
		}
	} else {
		e = &entry{}
		s.codes[email] = e
	}

	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}

	code := fmt.Sprintf("%06d", n.Int64())

	e.hash = hashCode(email, code)
	e.expires = now.Add(CodeTTL)
	e.attemptsLeft = MaxAttempts
	e.lastSent = now
	e.sends = append(e.sends, now)

	return code, nil
}

/*
function used to forget an email, it deletes the verification code entry of that email from the store.
it locks the store, so it is safe to call from anywhere. if the email has no entry, nothing happens.

note that the entry also holds the send history of the email (the cooldown and the hourly limit), so after
forgetting an email it can request a new code right away without waiting.
*/
func (s *Store) Forget(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.codes, email)
}

/*
function used to verify the 6 digit code that was sent to an email. it locks the store, finds the entry of
the email and checks the given code against the saved hash (the hashes are compared in constant time, so
the check does not leak timing information). if the code is correct, it returns a one time verification
token that the caller can use to prove the email was verified (for example during registration).

the checks, in this order:
	-> no entry for the email: fails with ErrNoCode.
	-> code expired: the attempts are set to 0 so the code can never be used again, and it fails with
	   ErrExpired.
	-> no attempts left: fails with ErrLocked.
	-> wrong code: one attempt is removed. if no attempts are left it fails with ErrLocked, otherwise it
	   fails with ErrWrongCode and returns how many attempts are left.

if the code is correct, the code is burned (attempts set to 0 and expiry set to now, so it cannot be used
twice), a random 32 byte hex token is generated and saved in the store with the email and an expiry of
TokenTTL from now.

returns:
	-> token, 0, nil on success
	-> "", attempts left, ErrWrongCode if the code is wrong and there are still attempts left
	-> "", 0, ErrNoCode / ErrExpired / ErrLocked if the code cannot be used
	-> "", 0, error if the random token could not be generated
*/
func (s *Store) Verify(email, code string) (string, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	e := s.codes[email]

	if e == nil {
		return "", 0, ErrNoCode
	}

	if now.After(e.expires) {
		e.attemptsLeft = 0
		return "", 0, ErrExpired
	}

	if e.attemptsLeft <= 0 {
		return "", 0, ErrLocked
	}

	given := hashCode(email, code)

	if subtle.ConstantTimeCompare(given[:], e.hash[:]) != 1 {
		e.attemptsLeft--

		if e.attemptsLeft <= 0 {
			return "", 0, ErrLocked
		}

		return "", e.attemptsLeft, ErrWrongCode
	}

	e.attemptsLeft = 0
	e.expires = now

	value, err := randomHex(32)
	if err != nil {
		return "", 0, err
	}

	s.tokens[value] = token{
		email:   email,
		expires: now.Add(TokenTTL),
	}

	return value, 0, nil
}

/*
function used to check that a verification token is valid for an email. the token is the one returned by
Verify after the user entered the correct code. it returns false if the token is empty, does not exist,
has expired (an expired token is also deleted from the store), or was issued for a different email. it
locks the store, so it is safe to call from anywhere.

note that it does NOT delete the token when the check succeeds, so the same token can be checked more than
once until it expires or DeleteToken is called. the caller should call DeleteToken after it finishes the
action the token was needed for (for example, after the registration succeeds).

returns:
	-> true if the token exists, has not expired and belongs to the given email
	-> false otherwise
*/
func (s *Store) CheckToken(value, email string) bool {
	if value == "" {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tokens[value]
	if !ok {
		return false
	}

	if time.Now().After(t.expires) {
		delete(s.tokens, value)
		return false
	}

	return t.email == email
}

/*
function used to delete a verification token from the store, so it can not be used again. it is meant to
be called after the token was used successfully (see CheckToken). it locks the store, so it is safe to call
from anywhere. if the token does not exist, nothing happens.
*/
func (s *Store) DeleteToken(value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tokens, value)
}
