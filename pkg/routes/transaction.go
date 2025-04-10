package routes

import (
	"net/http"
	"slices"

	"github.com/bakseter/spenn/pkg/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TransactionJSON struct {
	ID          uint   `json:"id,omitempty"`
	Amount      int    `json:"amount,string"`
	Description string `json:"description"`
	UserEmail   string `json:"user_email,omitempty"`
}

func PostTransaction(c *gin.Context, database *gorm.DB) {
	userInfo, err := getUserInfo(c)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})

		return
	}

	// Check if user exists in database
	var user models.User
	if err := database.Where("email = ?", userInfo.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create user if not exists
			user = models.User{Email: userInfo.Email}
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

	dbTransaction := models.Transaction{
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
}

func PatchTransaction(c *gin.Context, database *gorm.DB) {
	userInfo, err := getUserInfo(c)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})

		return
	}

	var user models.User
	if err := database.Where("email = ?", userInfo.Email).First(&user).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to fetch user, may not exist"})

		return
	}

	var transaction TransactionJSON
	if err := c.ShouldBindJSON(&transaction); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})

		return
	}

	transactionID := c.Param("id")
	if transactionID == "" {
		c.JSON(400, gin.H{"error": "id parameter not set"})

		return
	}

	var existingTransaction models.Transaction
	if err := database.Where("id = ?", transactionID).First(&existingTransaction).Error; err != nil {
		c.JSON(404, gin.H{"error": "transaction does not exist"})

		return
	}

	dbTransaction := models.Transaction{
		Amount:      transaction.Amount,
		Description: transaction.Description,
		UserID:      user.ID,
	}
	if err := database.UpdateColumns(&dbTransaction).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to save transaction"})

		return
	}

	c.Header("HX-Trigger", "reload-transactions")
	c.Status(http.StatusNoContent)
}

func GetAllTransactions(c *gin.Context, database *gorm.DB) {
	userInfo, err := getUserInfo(c)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})

		return
	}

	// Get user from database
	var user models.User
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

	var transactions []models.Transaction
	if err := database.Where("user_id = ?", user.ID).Find(&transactions).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to fetch transactions"})

		return
	}

	if len(transactions) == 0 {
		c.Header("Content-Type", "text/html")
		c.String(200, "<p class=\"italic\">Ingen transaksjoner</p>")

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

	slices.Reverse(transactionList)

	c.HTML(200, "transactions.html.tmpl", gin.H{
		"Transactions": transactionList,
		"Sum":          models.SumTransactions(transactions),
	})
}

func DeleteTransaction(c *gin.Context, database *gorm.DB) {
	userInfo, err := getUserInfo(c)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}

	var transaction models.Transaction
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
	var user models.User
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
}
