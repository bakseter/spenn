package api

import (
	"github.com/bakseter/spenn/pkg/config"
	"github.com/gin-contrib/cors"
)

func configureCORS(conf *config.Config) cors.Config {
	headers := []string{
		"Origin",
		"Content-Type",
		"Accept",
		"Authorization",
		"X-authentik-username",
		"X-authentik-groups",
		"X-authentik-entitlements",
		"X-authentik-email",
		"X-authentik-uid",
	}

	methods := []string{"GET", "PATCH", "PUT", "POST", "DELETE"}

	if conf.Local {
		return cors.Config{
			AllowOrigins: []string{"http://localhost:" + conf.Port},
			AllowMethods: methods,
			AllowHeaders: headers,
		}
	}

	return cors.Config{
		AllowOrigins: []string{conf.Host},
		AllowMethods: methods,
		AllowHeaders: headers,
	}
}
