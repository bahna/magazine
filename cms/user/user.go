// Package user provides an API for users and access management.
package user

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/mail"
	"time"

	"github.com/gorilla/securecookie"

	"gopkg.in/mgo.v2/bson"
)

var (
	ErrNotValid      = errors.New("invalid user")
	ErrWrongPassword = errors.New("wrong password")
	ErrPasswordMatch = errors.New("passwords do not match")
	ErrNotFound      = errors.New("user not found")
)

// User represents a user of a system.
type User struct {
	ID           bson.ObjectId `bson:"_id"`
	FirstName    string
	LastName     string
	Created      time.Time
	Active       bool
	Email        mail.Address
	PasswordHash []byte
	Roles        []Role
}

// Role represents a user's role. It's a mean for access control.
type Role int

//go:generate stringer -type=Role

const (
	_ Role = iota

	// Administrator is able to modify other users.
	Administrator

	// Author can enter tne /admin endpoint.
	Author

	// Visitor is any other user.
	Visitor

	// Expert is an expert in the Infocenter.
	Expert
)

// Roles collects all roles required for a project.
var Roles = []Role{
	Administrator,
	Author,
	Visitor,
	Expert,
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func init() {
	rand.Seed(time.Now().UnixNano())
}

// New returns a new user with all compulsory attributes.
func New(password, email, firstName, lastName string, roles []Role, key []byte) (u *User, err error) {
	passHash, err := MakeMAC([]byte(password), key)
	if err != nil {
		return
	}
	m, err := mail.ParseAddress(email)
	if err != nil {
		return
	}
	if len(roles) == 0 {
		return u, fmt.Errorf("user roles must be specified")
	}
	return &User{
		ID:           bson.NewObjectId(),
		FirstName:    firstName,
		LastName:     lastName,
		Created:      time.Now(),
		Active:       true,
		Email:        *m,
		PasswordHash: passHash,
		Roles:        roles,
	}, nil
}

// CheckMAC checks the message with a msgMAC using the provided key.
// Use it for a password check.
func CheckMAC(msg, msgMAC, key []byte) bool {
	mac := hmac.New(sha256.New, key)
	_, err := mac.Write(msg)
	if err != nil {
		log.Println("writing to mac failed")
	}
	wantMAC := mac.Sum(nil)
	return hmac.Equal(wantMAC, msgMAC)
}

// MakeMAC creates a HMAC from a message and a key.
func MakeMAC(msg, key []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, key)
	_, err := mac.Write(msg)
	if err != nil {
		return []byte{}, err
	}
	return mac.Sum(nil), nil
}

// Validate checks if all compulsory fields are present.
func Validate(u *User) bool {
	switch {
	case u.ID == bson.ObjectId(""):
		fallthrough
	case u.Email == mail.Address{}:
		fallthrough
	case len(u.PasswordHash) == 0:
		fallthrough
	case len(u.FirstName) == 0:
		fallthrough
	case len(u.LastName) == 0:
		return false
	}
	return true
}

// SetLoginCookie sets a secure cookie.
func SetLoginCookie(w http.ResponseWriter, u *User, c *securecookie.SecureCookie, t time.Duration) error {
	cookieValue := map[string]string{
		"id":    u.ID.Hex(),
		"email": u.Email.Address,
	}
	cookieName := "auth"
	path := "/"

	encoded, err := c.Encode(cookieName, cookieValue)
	if err != nil {
		return err
	}
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    encoded,
		Path:     path,
		Expires:  time.Now().Add(t),
		HttpOnly: true,
		// Secure: true, // TODO: must be turned on in production
	}
	http.SetCookie(w, cookie)
	return nil
}

// SetLogoutCookie partially implements user.Authenticator interface.
func SetLogoutCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    "auth",
		Value:   "",
		Path:    "/",
		Expires: time.Time{},
	})
}

// Verify implements user.Authenticator interface.
func Verify(password string, passwordHash, key []byte) bool {
	return CheckMAC([]byte(password), passwordHash, key)
}

// RandStringRunes returns a random string with the length of n.
func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
