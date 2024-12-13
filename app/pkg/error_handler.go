package pkg

import (
	"chess-engine/app/constant"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func PanicHandler(c *gin.Context) {
	if err := recover(); err != nil {
		str := fmt.Sprint(err)
		strArr := strings.Split(str, ":")

		key := strArr[0]
		msg := strings.Trim(strArr[1], " ")

		switch key {
		case
			constant.DataNotFound.GetResponseStatus():
			c.JSON(http.StatusBadRequest, BuildResponse_(key, msg, Null()))
			c.Abort()
		case
			constant.Unauthorized.GetResponseStatus():
			c.JSON(http.StatusUnauthorized, BuildResponse_(key, msg, Null()))
			c.Abort()
		default:
			c.JSON(http.StatusInternalServerError, BuildResponse_(key, msg, Null()))
			c.Abort()
		}
	}
}
