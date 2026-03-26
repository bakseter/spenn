package routes

import (
	"errors"
	"os"

	"github.com/gin-gonic/gin"
)

type TailscaleUser struct {
	Login string
	Name  string
}

func getTailscaleUser(ctx *gin.Context) (*TailscaleUser, error) {
	if os.Getenv("LOCAL") == "true" {
		return &TailscaleUser{
			Login: "test@github",
			Name:  "Test McTest",
		}, nil
	}

	login := ctx.GetHeader("Tailscale-User-Login")
	if login == "" {
		return nil, errors.New("missing Tailscale-User-Login header")
	}

	name := ctx.GetHeader("Tailscale-User-Name")
	if name == "" {
		return nil, errors.New("missing Tailscale-User-Name header")
	}

	return &TailscaleUser{
		Login: login,
		Name:  name,
	}, nil
}
