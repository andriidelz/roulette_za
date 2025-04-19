package oxapay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleWebhook processes incoming webhook events from OxaPay
func (c *Client) handleWebhook(ctx *gin.Context) {
	// Read and parse the request body
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Failed to read request body"})
		return
	}

	// Verify webhook signature if webhook key is provided
	if c.webhookKey != "" {
		signature := ctx.GetHeader("X-OxaPay-Signature")
		if !c.verifyWebhookSignature(body, signature) {
			ctx.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Invalid signature"})
			return
		}
	}

	// Parse webhook event
	var webhookEvent WebhookEvent
	if err := json.Unmarshal(body, &webhookEvent); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid JSON payload"})
		return
	}

	// Store raw payload
	webhookEvent.RawPayload = string(body)
	webhookEvent.Verified = c.webhookKey != ""

	// Save webhook event to database
	if c.db != nil {
		if err := c.db.Create(&webhookEvent).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Failed to save webhook event"})
			return
		}

		// Update payout status
		if err := c.UpdatePayoutStatus(&webhookEvent); err != nil {
			// Log the error but don't fail the request
			fmt.Printf("Error updating payout status: %v\n", err)
		}
	}

	// Return success
	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}

// verifyWebhookSignature verifies the signature of a webhook event
func (c *Client) verifyWebhookSignature(payload []byte, signature string) bool {
	if c.webhookKey == "" || signature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(c.webhookKey))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedMAC))
}

// UpdatePayoutStatus updates the status of a payout based on webhook event
func (c *Client) UpdatePayoutStatus(webhookEvent *WebhookEvent) error {
	if c.db == nil {
		return fmt.Errorf("database connection is not configured")
	}

	// Найти payout по ID
	var payout Payout
	if err := c.db.Where("id = ?", webhookEvent.PayoutID).First(&payout).Error; err != nil {
		return fmt.Errorf("payout not found: %w", err)
	}

	// Обновить статус
	payout.Status = webhookEvent.Status
	payout.LastWebhookEvent = webhookEvent

	// Если есть хеш транзакции, обновить его
	if webhookEvent.Transaction != "" {
		payout.TransactionHash = webhookEvent.Transaction
	}

	// Обновить запись в БД
	if err := c.db.Save(&payout).Error; err != nil {
		return fmt.Errorf("failed to update payout: %w", err)
	}

	return nil
}
