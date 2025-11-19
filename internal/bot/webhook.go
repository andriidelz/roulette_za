package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"roulette/internal/logger"

	"github.com/mymmrac/telego"
)

// WebhookServer handles incoming webhook requests from Telegram
type WebhookServer struct {
	bot         *Bot
	server      *http.Server
	webhookURL  string
	webhookPath string // Path for webhook URL (e.g., /webhook)
	secretToken string // Secret token for validation via X-Telegram-Bot-Api-Secret-Token header
}

// NewWebhookServer creates a new webhook server
func NewWebhookServer(bot *Bot, listenAddr, webhookURL, webhookPath, secretToken string) *WebhookServer {
	ws := &WebhookServer{
		bot:         bot,
		webhookURL:  webhookURL,
		webhookPath: webhookPath,
		secretToken: secretToken,
	}

	mux := http.NewServeMux()
	mux.HandleFunc(webhookPath, ws.handleWebhook)
	mux.HandleFunc("/health", ws.handleHealthCheck)

	ws.server = &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return ws
}

// Start starts the webhook server and sets up the webhook with Telegram
func (ws *WebhookServer) Start() error {
	// Delete any existing webhook first
	if err := ws.bot.bot.DeleteWebhook(ws.bot.ctx, &telego.DeleteWebhookParams{
		DropPendingUpdates: false,
	}); err != nil {
		logger.Warning.Printf("Failed to delete existing webhook: %v", err)
	}

	// Set up the webhook with Telegram
	fullWebhookURL := ws.webhookURL + ws.webhookPath
	if err := ws.bot.bot.SetWebhook(ws.bot.ctx, &telego.SetWebhookParams{
		URL:                fullWebhookURL,
		MaxConnections:     100,
		DropPendingUpdates: false,
		SecretToken:        ws.secretToken, // Telegram will send this in X-Telegram-Bot-Api-Secret-Token header
	}); err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}

	logger.Info.Printf("Webhook set successfully: %s (secret token configured)", fullWebhookURL)

	// Start the HTTP server
	go func() {
		logger.Info.Printf("Starting webhook server on %s", ws.server.Addr)
		if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error.Printf("Webhook server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully stops the webhook server
func (ws *WebhookServer) Stop() error {
	// Delete webhook from Telegram
	if err := ws.bot.bot.DeleteWebhook(ws.bot.ctx, &telego.DeleteWebhookParams{
		DropPendingUpdates: false,
	}); err != nil {
		logger.Warning.Printf("Failed to delete webhook on stop: %v", err)
	}

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ws.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown webhook server: %w", err)
	}

	logger.Info.Println("Webhook server stopped")
	return nil
}

// handleWebhook processes incoming webhook requests from Telegram
func (ws *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate secret token from header if configured
	if ws.secretToken != "" {
		receivedToken := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
		if receivedToken != ws.secretToken {
			logger.Warning.Printf("Invalid secret token received from %s", r.RemoteAddr)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error.Printf("Failed to read webhook body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse update
	var update telego.Update
	if err := json.Unmarshal(body, &update); err != nil {
		logger.Error.Printf("Failed to parse webhook update: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Process update
	ws.bot.processUpdate(update)

	// Respond with 200 OK
	w.WriteHeader(http.StatusOK)
}

// handleHealthCheck provides a health check endpoint
func (ws *WebhookServer) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
