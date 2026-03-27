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
	Login       string `json:"login,omitempty"`
	Category    string `json:"category,omitempty"`
	Shared      string `json:"shared,omitempty"`
	Date        string `json:"date,omitempty"`
}

func TransactionRoutes(router *gin.RouterGroup, database *gorm.DB) {
	router.GET("/transactions", models.WithDatabase(getAllTransactions, database))
	router.POST("/transaction", models.WithDatabase(postTransaction, database))
	router.DELETE("/transaction/:id", models.WithDatabase(deleteTransaction, database))
}

func postTransaction(ctx *gin.Context, database *gorm.DB) {
	tailscaleUser, err := getTailscaleUser(ctx)
	if err != nil {
		ctx.JSON(401, gin.H{"error": err.Error()})

		return
	}

	// Check if user exists in database
	var user models.User
	if err := database.Where("login = ?", tailscaleUser.Login).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create user if not exists
			user = models.User{Login: tailscaleUser.Login}
			if err := database.Create(&user).Error; err != nil {
				ctx.JSON(500, gin.H{"error": "failed to create user"})

				return
			}
		} else {
			ctx.JSON(500, gin.H{"error": "failed to fetch user"})

			return
		}
	}

	// User exists, proceed with transaction
	var transaction TransactionJSON
	if err := ctx.ShouldBindJSON(&transaction); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})

		return
	}

	dbTransaction := models.Transaction{
		Amount:      transaction.Amount,
		Description: transaction.Description,
		UserID:      user.ID,
		Category:    transaction.Category,
		Shared:      transaction.Shared == "on",
	}
	if err := database.Create(&dbTransaction).Error; err != nil {
		ctx.JSON(500, gin.H{"error": "failed to save transaction"})

		return
	}

	ctx.Header("HX-Trigger", "reload-transactions")
	ctx.Status(http.StatusNoContent)
}

func getAllTransactions(ctx *gin.Context, database *gorm.DB) {
	tailscaleUser, err := getTailscaleUser(ctx)
	if err != nil {
		ctx.JSON(401, gin.H{"error": err.Error()})

		return
	}

	// Get user from database
	var user models.User
	if err := database.Where("login = ?", tailscaleUser.Login).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.Header("Content-Type", "text/html")
			ctx.String(200, "<p class=\"italic\">Ingen transaksjoner</p>")

			return
		} else {
			ctx.JSON(500, gin.H{"error": "failed to fetch user"})

			return
		}
	}

	var transactions []models.Transaction
	if err := database.Where("user_id = ?", user.ID).Find(&transactions).Error; err != nil {
		ctx.JSON(500, gin.H{"error": "failed to fetch transactions"})

		return
	}

	if len(transactions) == 0 {
		ctx.Header("Content-Type", "text/html")
		ctx.String(200, "<p class=\"italic\">Ingen transaksjoner</p>")

		return
	}

	shared := ctx.Query("shared")

	var transactionList []TransactionJSON
	for _, transaction := range transactions {
		if (shared == "on" && transaction.Shared) ||
			(shared == "off" && !transaction.Shared) ||
			(shared == "") {
			transactionList = append(transactionList, TransactionJSON{
				ID:          transaction.ID,
				Login:       tailscaleUser.Login,
				Amount:      transaction.Amount,
				Description: transaction.Description,
				Category:    transaction.Category,
				Shared: func() string {
					if transaction.Shared {
						return "on"
					}

					return "off"
				}(),
				Date: transaction.CreatedAt.Format("02.01.2006"),
			})
		}
	}

	slices.Reverse(transactionList)

	ctx.HTML(200, "transactions.html.tmpl", gin.H{
		"Transactions": transactionList,
		"Sum":          models.SumTransactions(transactions),
	})
}

/*
func patchTransaction(c *gin.Context, database *gorm.DB) {
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
*/

func deleteTransaction(ctx *gin.Context, database *gorm.DB) {
	tailscaleUser, err := getTailscaleUser(ctx)
	if err != nil {
		ctx.JSON(401, gin.H{"error": err.Error()})

		return
	}

	var transaction models.Transaction
	if err := database.Where("id = ?", ctx.Param("id")).First(&transaction).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(404, gin.H{"error": "transaction not found"})

			return
		} else {
			ctx.JSON(500, gin.H{"error": "failed to fetch transaction"})

			return
		}
	}

	// Check if the user owns the transaction
	var user models.User
	if err := database.Where("id = ?", transaction.UserID).First(&user).Error; err != nil {
		ctx.JSON(500, gin.H{"error": "failed to fetch user"})

		return
	}

	if user.Login != tailscaleUser.Login {
		ctx.JSON(403, gin.H{"error": "forbidden"})

		return
	}

	// Delete the transaction
	if err := database.Delete(&transaction).Error; err != nil {
		ctx.JSON(500, gin.H{"error": "failed to delete transaction"})

		return
	}

	ctx.Header("HX-Trigger", "reload-transactions")
	ctx.Status(http.StatusNoContent)
}
