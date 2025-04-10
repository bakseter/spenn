package main

import (
	"log"

	"github.com/bakseter/spenn/pkg/models"
	"github.com/bakseter/spenn/pkg/routes"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func main() {
	router := gin.Default()
	router.Use(static.Serve("/", static.LocalFile("./static", true)))
	router.LoadHTMLGlob("templates/*")

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
