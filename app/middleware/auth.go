package middleware

import (
	"chess-engine/app/constant"
	"chess-engine/app/domain/dao"
	"chess-engine/app/pkg"
	"chess-engine/app/repository"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// authUserKey is the gin context key holding the authenticated user.
const authUserKey = "authUser"

// RequireAuth resolves the bearer token into a user and aborts with 401 if it
// does not match one. Routes that mutate or expose a specific user's data must
// sit behind this: PUT and DELETE /api/user/:userID previously accepted any
// caller and let anyone delete any account by guessing an integer id.
func RequireAuth(userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c)
		if token == "" {
			abortUnauthorized(c, "missing bearer token")
			return
		}

		user, err := userRepo.FindUserByToken(token)
		if err != nil || user.ID == 0 {
			abortUnauthorized(c, "invalid token")
			return
		}

		c.Set(authUserKey, user)
		c.Next()
	}
}

// AuthUser returns the user resolved by RequireAuth. ok is false if the route
// was not behind the middleware.
func AuthUser(c *gin.Context) (dao.User, bool) {
	v, exists := c.Get(authUserKey)
	if !exists {
		return dao.User{}, false
	}
	user, ok := v.(dao.User)
	return user, ok
}

// bearerToken reads the token from "Authorization: Bearer <token>", falling back
// to the X-Auth-Token header.
func bearerToken(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); h != "" {
		if after, found := cutPrefixFold(h, "bearer "); found {
			return strings.TrimSpace(after)
		}
	}
	return strings.TrimSpace(c.GetHeader("X-Auth-Token"))
}

func cutPrefixFold(s, prefix string) (string, bool) {
	if len(s) < len(prefix) || !strings.EqualFold(s[:len(prefix)], prefix) {
		return "", false
	}
	return s[len(prefix):], true
}

func abortUnauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, pkg.BuildResponse_(
		constant.Unauthorized.GetResponseStatus(), msg, pkg.Null(),
	))
}
