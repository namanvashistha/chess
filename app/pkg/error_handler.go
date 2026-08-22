package pkg

import (
	"chess-engine/app/constant"
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

// PanicHandler recovers a panic raised by PanicException and turns it into a
// JSON error response.
//
// It must also survive panics it did not raise (nil dereferences, index errors,
// third-party panics). Those have no "KEY: message" shape, so the key/message
// split is done defensively: splitting on ":" and indexing [1] unconditionally
// used to panic *inside* the recover on any panic value without a colon, which
// takes down the whole process instead of returning a 500.
func PanicHandler(c *gin.Context) {
	rec := recover()
	if rec == nil {
		return
	}

	str := fmt.Sprint(rec)
	key, msg := str, ""
	if idx := strings.IndexByte(str, ':'); idx >= 0 {
		key = str[:idx]
		msg = strings.TrimSpace(str[idx+1:])
	}

	switch key {
	case constant.DataNotFound.GetResponseStatus():
		c.AbortWithStatusJSON(http.StatusBadRequest, BuildResponse_(key, msg, Null()))
	case constant.InvalidRequest.GetResponseStatus():
		c.AbortWithStatusJSON(http.StatusBadRequest, BuildResponse_(key, msg, Null()))
	case constant.Unauthorized.GetResponseStatus():
		c.AbortWithStatusJSON(http.StatusUnauthorized, BuildResponse_(key, msg, Null()))
	case constant.UnknownError.GetResponseStatus():
		c.AbortWithStatusJSON(http.StatusInternalServerError, BuildResponse_(key, msg, Null()))
	default:
		// Not one of ours: a real bug. Log it with the panic value and return a
		// generic 500 rather than echoing internals back to the client.
		log.Errorf("unhandled panic in %s %s: %v", c.Request.Method, c.Request.URL.Path, rec)
		c.AbortWithStatusJSON(http.StatusInternalServerError, BuildResponse_(
			constant.UnknownError.GetResponseStatus(),
			constant.UnknownError.GetResponseMessage(),
			Null(),
		))
	}
}
