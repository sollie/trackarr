package ws

import (
	"github.com/labstack/echo/v5"
)

func HandlerFunc(context *echo.Context) error {
	return m.HandleRequest(context.Response(), context.Request())
}
