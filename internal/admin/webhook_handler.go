package admin

import (
	"fmt"
	"net/http"
	"roulette/internal/payment"
	"roulette/internal/service"

	"github.com/gin-gonic/gin"
)

// setupPaymentWebhooks настраивает обработчики вебхуков для платежных систем
func (a *AdminPanel) setupPaymentWebhooks(paymentService *service.PaymentService) {
	// Создаем группу маршрутов для вебхуков
	webhooks := a.router.Group("/webhooks/payment")
	{
		// Обработчик вебхуков OxaPay
		webhooks.POST("/oxapay", a.handleOxaPayWebhook(paymentService))

		// Здесь можно добавить обработчики для других платежных систем
		// webhooks.POST("/otherprovider", a.handleOtherProviderWebhook(paymentService))
	}
}

// handleOxaPayWebhook обрабатывает вебхуки от OxaPay
func (a *AdminPanel) handleOxaPayWebhook(paymentService *service.PaymentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Здесь мы не обрабатываем напрямую вебхук OxaPay,
		// так как эта логика уже реализована в клиенте OxaPay
		// Клиент OxaPay сам обновит статусы в своей базе данных

		// Для интеграции с нашей системой мы можем использовать
		// дополнительный обработчик после успешной обработки вебхука

		// Примечание: Обычно это делается через события или хуки в клиенте OxaPay
		// или путем периодического обновления статусов из базы данных OxaPay

		// Здесь просто возвращаем успешный ответ, фактическая обработка
		// происходит внутри клиента OxaPay
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	}
}

// Этот метод может быть вызван из клиента OxaPay после обработки вебхука
// или из отдельной горутины, которая синхронизирует статусы
func (a *AdminPanel) updateWithdrawalFromWebhook(
	paymentService *service.PaymentService,
	providerName string,
	withdrawalID string,
	status payment.WithdrawalStatus,
	transactionHash string,
) {
	err := paymentService.ProcessWebhookUpdate(providerName, withdrawalID, status, transactionHash)
	if err != nil {
		fmt.Printf("Error updating withdrawal from webhook: %v\n", err)
	}
}
