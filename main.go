package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type TransactionJSON struct {
	ID          uint   `json:"id,omitempty"`
	Amount      int    `json:"amount,string"`
	Description string `json:"description"`
	UserEmail   string `json:"user_email,omitempty"`
}

type User struct {
	gorm.Model
	Email        string
	Transactions []Transaction
}

type Transaction struct {
	gorm.Model
	Amount      int
	Description string
	UserID      uint
}

func main() {
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

	database.AutoMigrate(&User{}, &Transaction{})

	api := router.Group("/api")
	{
		api.GET("/status", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "ok",
			})
		})

		api.POST("/transaction", func(c *gin.Context) {
			userInfo, err := getUserInfo(c)
			if err != nil {
				c.JSON(401, gin.H{"error": err.Error()})
				return
			}

			// Check if user exists in database
			var user User
			if err := database.Where("email = ?", userInfo.Email).First(&user).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					// Create user if not exists
					user = User{Email: userInfo.Email}
					if err := database.Create(&user).Error; err != nil {
						c.JSON(500, gin.H{"error": "failed to create user"})
						return
					}
				} else {
					c.JSON(500, gin.H{"error": "failed to fetch user"})
					return
				}
			}

			// User exists, proceed with transaction
			var transaction TransactionJSON
			if err := c.ShouldBindJSON(&transaction); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			dbTransaction := Transaction{
				Amount:      transaction.Amount,
				Description: transaction.Description,
				UserID:      user.ID,
			}
			if err := database.Create(&dbTransaction).Error; err != nil {
				c.JSON(500, gin.H{"error": "failed to save transaction"})
				return
			}

			c.Header("HX-Trigger", "reload-transactions")
			c.Status(http.StatusNoContent)
		})

		api.GET("/transactions", func(c *gin.Context) {
			userInfo, err := getUserInfo(c)
			if err != nil {
				c.JSON(401, gin.H{"error": err.Error()})
				return
			}

			// Get user from database
			var user User
			if err := database.Where("email = ?", userInfo.Email).First(&user).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					c.Header("Content-Type", "text/html")
					c.String(200, "<p class=\"italic\">Ingen transaksjoner</p>")
					return
				} else {
					c.JSON(500, gin.H{"error": "failed to fetch user"})
					return
				}
			}

			var transactions []Transaction
			if err := database.Where("user_id = ?", user.ID).Find(&transactions).Error; err != nil {
				c.JSON(500, gin.H{"error": "failed to fetch transactions"})
				return
			}

			if len(transactions) == 0 {
				c.Header("Content-Type", "text/html")
				c.String(200, "<p class=\"italic\">No transactions found</p>")
				return
			}

			var transactionList []TransactionJSON
			for _, transaction := range transactions {
				transactionList = append(transactionList, TransactionJSON{
					ID:          transaction.ID,
					UserEmail:   userInfo.Email,
					Amount:      transaction.Amount,
					Description: transaction.Description,
				})
			}

			c.HTML(200, "transactions.html.tmpl", gin.H{
				"Transactions": transactionList,
				"Sum":          sumTransactions(transactions),
			})
		})

		api.DELETE("/transaction/:id", func(c *gin.Context) {
			userInfo, err := getUserInfo(c)
			if err != nil {
				c.JSON(401, gin.H{"error": err.Error()})
				return
			}

			var transaction Transaction
			if err := database.Where("id = ?", c.Param("id")).First(&transaction).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					c.JSON(404, gin.H{"error": "transaction not found"})
					return
				} else {
					c.JSON(500, gin.H{"error": "failed to fetch transaction"})
					return
				}
			}

			// Check if the user owns the transaction
			var user User
			if err := database.Where("id = ?", transaction.UserID).First(&user).Error; err != nil {
				c.JSON(500, gin.H{"error": "failed to fetch user"})
				return
			}

			if user.Email != userInfo.Email {
				c.JSON(403, gin.H{"error": "forbidden"})
				return
			}

			// Delete the transaction
			if err := database.Delete(&transaction).Error; err != nil {
				c.JSON(500, gin.H{"error": "failed to delete transaction"})
				return
			}

			c.Header("HX-Trigger", "reload-transactions")
			c.Status(http.StatusNoContent)
		})
	}

	router.Run(":8080")
}

func sumTransactions(transactions []Transaction) int {
	sum := 0

	for _, transaction := range transactions {
		sum += transaction.Amount
	}

	return sum
}

type UserInfo struct {
	User  string `json:"user"`
	Email string `json:"email"`
}

func getUserInfo(c *gin.Context) (*UserInfo, error) {
	if os.Getenv("DEV") == "true" {
		return &UserInfo{
			User:  "test",
			Email: "test@example.com",
		}, nil
	}

	oauth2UserinfoEndpoint := os.Getenv("OAUTH2_USERINFO_ENDPOINT")
	if oauth2UserinfoEndpoint == "" {
		return nil, errors.New("OAUTH2_USERINFO_ENDPOINT is not set")
	}

	cookie, err := c.Cookie("_oauth2_proxy")
	if err != nil {
		return nil, err
	}

	// Send cookie to auth endpoint to get userinfo
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", oauth2UserinfoEndpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Cookie", "_oauth2_proxy="+cookie)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, err
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}
