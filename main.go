package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type TransactionJSON struct {
	Amount      int    `json:"amount,string"`
	Description string `json:"description"`
}

type TransactionDB struct {
	gorm.Model
	Amount      int
	Description string
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	router.Use(static.Serve("/", static.LocalFile("./static", true)))
    router.LoadHTMLGlob("templates/*")

	databaseHost := os.Getenv("DATABASE_HOST")
	if databaseHost == "" {
		databaseHost = "localhost"
	}

	databaseUsername := os.Getenv("DATABASE_USERNAME")
	if databaseUsername == "" {
		panic("DATABASE_USERNAME is not set")
	}

	databasePassword := os.Getenv("DATABASE_PASSWORD")
	if databasePassword == "" {
		panic("DATABASE_PASSWORD is not set")
	}

	databaseName := os.Getenv("DATABASE_NAME")
	if databaseName == "" {
		panic("DATABASE_NAME is not set")
	}

	dataSourceName := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable TimeZone=Europe/Oslo",
        databaseHost,
		databaseUsername,
		databasePassword,
		databaseName,
	)

	database, err := gorm.Open(postgres.Open(dataSourceName), &gorm.Config{})
	if err != nil {
        fmt.Errorf("failed to connect to database: %v", err)
	}

	database.AutoMigrate(&TransactionDB{})

	api := router.Group("/api")
	{
		api.GET("/status", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "ok",
			})
		})

		api.POST("/transaction", func(c *gin.Context) {
			var transaction TransactionJSON
			if err := c.ShouldBindJSON(&transaction); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			dbTransaction := TransactionDB{
				Amount:      transaction.Amount,
				Description: transaction.Description,
			}
			if err := database.Create(&dbTransaction).Error; err != nil {
				c.JSON(500, gin.H{"error": "failed to save transaction"})
				return
			}

			c.Header("Content-Type", "text/html")
			c.String(200, "<p>transaction received :)</p>")
		})

		api.GET("/transactions", func(c *gin.Context) {
			var transactions []TransactionDB
			if err := database.Find(&transactions).Error; err != nil {
				c.JSON(500, gin.H{"error": "failed to fetch transactions"})
				return
			}

			if len(transactions) == 0 {
				c.JSON(200, gin.H{"message": "no transactions found"})
				return
			}

			var transactionList []TransactionJSON
			for _, transaction := range transactions {
				transactionList = append(transactionList, TransactionJSON{
					Amount:      transaction.Amount,
					Description: transaction.Description,
				})
			}

			c.HTML(200, "transactions.html.tmpl", gin.H{
                "Transactions": transactionList,
            })
		})
	}

	router.Run(":8080")
}
