package middleware

import (
	"chess-engine/app/domain/dao"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// stubUserRepo resolves exactly one token.
type stubUserRepo struct{ validToken string }

func (s stubUserRepo) FindUserByToken(token string) (dao.User, error) {
	if token == s.validToken {
		return dao.User{ID: 42, Name: "owner", Token: token}, nil
	}
	return dao.User{}, errors.New("not found")
}

func (s stubUserRepo) FindAllUser() ([]dao.User, error)      { return nil, nil }
func (s stubUserRepo) FindUserById(id int) (dao.User, error) { return dao.User{}, nil }
func (s stubUserRepo) Save(u *dao.User) (dao.User, error)    { return *u, nil }
func (s stubUserRepo) DeleteUserById(id int) error           { return nil }

func newTestRouter(repo stubUserRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", RequireAuth(repo), func(c *gin.Context) {
		u, ok := AuthUser(c)
		if !ok {
			c.String(http.StatusInternalServerError, "no user in context")
			return
		}
		c.String(http.StatusOK, u.Name)
	})
	return r
}

func TestRequireAuth(t *testing.T) {
	const good = "valid-token"
	r := newTestRouter(stubUserRepo{validToken: good})

	cases := []struct {
		name     string
		header   string
		value    string
		wantCode int
	}{
		// Regression: PUT/DELETE /api/user/:userID accepted unauthenticated
		// callers, so anyone could delete any account by guessing an integer.
		{"no header", "", "", http.StatusUnauthorized},
		{"bad token", "Authorization", "Bearer nope", http.StatusUnauthorized},
		{"empty bearer", "Authorization", "Bearer ", http.StatusUnauthorized},
		{"wrong scheme", "Authorization", good, http.StatusUnauthorized},
		{"valid bearer", "Authorization", "Bearer " + good, http.StatusOK},
		{"case-insensitive scheme", "Authorization", "bearer " + good, http.StatusOK},
		{"x-auth-token fallback", "X-Auth-Token", good, http.StatusOK},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if c.header != "" {
				req.Header.Set(c.header, c.value)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != c.wantCode {
				t.Errorf("status = %d, want %d (body %q)", w.Code, c.wantCode, w.Body.String())
			}
			if c.wantCode == http.StatusOK && w.Body.String() != "owner" {
				t.Errorf("body = %q, want the authenticated user's name", w.Body.String())
			}
		})
	}
}
