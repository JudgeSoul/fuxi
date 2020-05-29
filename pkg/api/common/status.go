package common

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

const (
	PageKey  = "page"
	PageSize = "pageSize"
)

const (
	msg    = "message"
	data   = "data"
	status = "errors"
	code   = "code"
)

func ToRequestParamsError(g *gin.Context, err error) {
	g.JSON(
		http.StatusBadRequest,
		gin.H{
			code:   http.StatusBadRequest,
			data:   "",
			msg:    "Request bad error",
			status: err.Error(),
		},
	)
}

func ToInternalServerError(g *gin.Context, runtimeData interface{}, err error) {
	g.JSON(
		http.StatusInternalServerError,
		gin.H{
			code:   http.StatusInternalServerError,
			data:   runtimeData,
			msg:    "",
			status: err.Error(),
		},
	)
}