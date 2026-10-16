package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"chess-engine/app/store"

	petname "github.com/dustinkirkland/golang-petname"
)

// Sessions are anonymous: POST /api/session makes a user with a random name
// and returns a token. Browsers get it as an HttpOnly cookie; native apps send
// it as "Authorization: Bearer <token>". Only the token's SHA-256 is stored.

const (
	cookieName   = "session"
	cookieMaxAge = 365 * 24 * time.Hour
	maxNameLen   = 24
)

func newToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func requestToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	if c, err := r.Cookie(cookieName); err == nil {
		return c.Value
	}
	return ""
}

// user returns the signed-in user, or nil.
func (s *Server) user(r *http.Request) (*store.User, error) {
	token := requestToken(r)
	if token == "" {
		return nil, nil
	}
	u, err := s.cfg.Store.UserByTokenHash(r.Context(), hashToken(token))
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// requireUser returns the signed-in user, or writes a 401 and returns nil.
func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) *store.User {
	u, err := s.user(r)
	if err != nil {
		s.internal(w, "load user", err)
		return nil
	}
	if u == nil {
		writeError(w, http.StatusUnauthorized, "signed_out", "start a session first")
	}
	return u
}

func (s *Server) setCookie(w http.ResponseWriter, r *http.Request, token string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteLaxMode,
	})
}

var bigTenThousand = big.NewInt(10000)

func randomName() string {
	n, _ := rand.Int(rand.Reader, bigTenThousand)
	return fmt.Sprintf("%s-%04d", petname.Generate(2, "-"), n.Int64())
}

// sessionResponse is what POST /api/session returns. The token is in the body
// for native clients; browsers can ignore it and use the cookie.
type sessionResponse struct {
	User  store.User `json:"user"`
	Token string     `json:"token"`
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	token := newToken()
	u, err := s.cfg.Store.CreateUser(r.Context(), randomName(), hashToken(token))
	if err != nil {
		s.internal(w, "create user", err)
		return
	}
	s.setCookie(w, r, token, cookieMaxAge)
	writeJSON(w, http.StatusCreated, sessionResponse{User: u, Token: token})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	if u := s.requireUser(w, r); u != nil {
		writeJSON(w, http.StatusOK, u)
	}
}

func (s *Server) rename(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &body) {
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" || utf8.RuneCountInString(name) > maxNameLen {
		writeError(w, http.StatusBadRequest, "bad_name", fmt.Sprintf("names are 1 to %d characters", maxNameLen))
		return
	}
	updated, err := s.cfg.Store.RenameUser(r.Context(), u.ID, name)
	if err != nil {
		s.internal(w, "rename user", err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteMe(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	if err := s.cfg.Store.DeleteUser(r.Context(), u.ID); err != nil {
		s.internal(w, "delete user", err)
		return
	}
	s.setCookie(w, r, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) internal(w http.ResponseWriter, what string, err error) {
	s.cfg.Log.Error(what, "err", err)
	writeError(w, http.StatusInternalServerError, "internal", "something went wrong, please try again")
}
