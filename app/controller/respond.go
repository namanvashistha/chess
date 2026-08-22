package controller

import (
	"chess-engine/app/constant"
	"chess-engine/app/pkg"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// respond writes a service result as the API's standard envelope.
//
// This is the only place that knows about HTTP status codes. Services return
// (value, error) and never touch the gin context, which is what makes them
// testable without spinning up a request.
func respond(c *gin.Context, data interface{}, err error) {
	if err == nil {
		c.JSON(pkg.HTTPStatus(constant.Success), pkg.BuildResponse(constant.Success, data))
		return
	}

	if appErr, ok := pkg.AsAppError(err); ok {
		if appErr.Err != nil {
			log.Errorf("%s %s: %v", c.Request.Method, c.Request.URL.Path, appErr)
		}
		c.JSON(pkg.HTTPStatus(appErr.Status), pkg.BuildResponse_(
			appErr.Status.GetResponseStatus(), appErr.Msg, pkg.Null(),
		))
		return
	}

	// Not a classified error: a bug. Log it and say nothing useful to the client.
	log.Errorf("unhandled error in %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	c.JSON(pkg.HTTPStatus(constant.UnknownError), pkg.BuildResponse_(
		constant.UnknownError.GetResponseStatus(),
		constant.UnknownError.GetResponseMessage(),
		pkg.Null(),
	))
}
