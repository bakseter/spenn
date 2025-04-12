package main

import (
	"log"
	"os"
	"time"

	"github.com/bakseter/spenn/pkg/models"
	"github.com/bakseter/spenn/pkg/routes"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func main() {
	dev := func() bool {
		dev_ := os.Getenv("DEV")

		return dev_ == "true"
	}()

	router := gin.Default()
	router.Use(static.Serve("/", static.LocalFile("./static", true)))
	router.LoadHTMLGlob("templates/*")

	if !dev {
		gin.SetMode(gin.ReleaseMode)
	}

	if !dev {
		host := os.Getenv("HOST")
		router.Use(cors.New(cors.Config{
			AllowOrigins: []string{host},
			AllowMethods: []string{"GET", "PUT", "POST", "DELETE"},
			AllowHeaders: []string{
				"Origin",
				"Content-Type",
				"Accept",
				"Authorization",
				"X-Auth-Request-User",
				"X-Auth-Request-Email",
				"X-Auth-Requiest-Groups",
				"X-Auth-Request-Access-Token",
				"X-Auth-Request-Preferred-Username",
				"X-Forwarded-Access-Token",
				"X-Forwarded-User",
				"X-Forwarded-Email",
				"X-Forwarded-Preferred-Username",
				"X-Forwarded-Groups",
			},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
	}

	database, err := models.InitializeDatabase()
	if err != nil {
		log.Fatal(err)
	}

	err = database.AutoMigrate(&models.User{}, &models.Transaction{})
	if err != nil {
		log.Fatal(err)
	}

	api := router.Group("/api")
	{
		api.GET("/status", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "ok",
			})
		})

		api.GET("/transactions", withDatabase(routes.GetAllTransactions, database))
		api.POST("/transaction", withDatabase(routes.PostTransaction, database))
		api.DELETE("/transaction/:id", withDatabase(routes.DeleteTransaction, database))
	}

	err = router.Run(":8080")
	if err != nil {
		log.Fatal(err)
	}
}

func withDatabase(fn func(*gin.Context, *gorm.DB), database *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		fn(c, database)
	}
}
