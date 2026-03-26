package api

import (
	"github.com/bakseter/spenn/pkg/config"
	"github.com/bakseter/spenn/pkg/models"
	"github.com/bakseter/spenn/pkg/routes"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func Start(conf *config.Config, log *logrus.Logger) error {
	router := gin.New()

	router.Use(config.LogrusMiddleware(log))
	router.Use(gin.Recovery())
	router.Use(config.MetricsMiddleware(conf))
	router.Use(cors.New(configureCORS(conf)))
	router.Use(static.Serve("/", static.LocalFile("./static", true)))

	router.LoadHTMLGlob("templates/*")

	err := router.SetTrustedProxies(nil)
	if err != nil {
		return err
	}

	if !conf.Local {
		gin.SetMode(gin.ReleaseMode)
	}

	database, err := models.ConfigureDatabase(conf)
	if err != nil {
		return err
	}

	addRoutes(router, database)

	err = router.Run(":" + conf.Port)
	if err != nil {
		return err
	}

	return nil
}

func addRoutes(router *gin.Engine, database *gorm.DB) {
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := router.Group("/api")
	{
		api.GET("/status", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "ok",
			})
		})

		routes.TransactionRoutes(api, database)
	}
}
