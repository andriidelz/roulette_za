package bot

import (
	"fmt"
	"time"

	"roulette/internal/logger"

	"github.com/goccy/go-json"
	"github.com/mymmrac/telego"
	"github.com/valyala/fasthttp"
)

// WebhookServer handles incoming webhook requests from Telegram using fasthttp
type WebhookServer struct {
	bot         *Bot
	server      *fasthttp.Server
	webhookURL  string
	webhookPath string
	secretToken string
	listenAddr  string
}

// NewWebhookServer creates a new webhook server
func NewWebhookServer(bot *Bot, listenAddr, webhookURL, webhookPath, secretToken string) *WebhookServer {
	ws := &WebhookServer{
		bot:         bot,
		webhookURL:  webhookURL,
		webhookPath: webhookPath,
		secretToken: secretToken,
		listenAddr:  listenAddr,
	}

	ws.server = &fasthttp.Server{
		Handler:            ws.requestHandler,
		ReadTimeout:        10 * time.Second,
		WriteTimeout:       10 * time.Second,
		IdleTimeout:        60 * time.Second,
		MaxRequestBodySize: 10 * 1024 * 1024, // 10MB
		ReduceMemoryUsage:  true,
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
		SecretToken:        ws.secretToken,
	}); err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}

	logger.Info.Printf("Webhook set successfully: %s (secret token configured)", fullWebhookURL)

	// Start the fasthttp server
	go func() {
		logger.Info.Printf("Starting webhook server on %s (fasthttp)", ws.listenAddr)
		if err := ws.server.ListenAndServe(ws.listenAddr); err != nil {
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

	// Shutdown fasthttp server
	if err := ws.server.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown webhook server: %w", err)
	}

	logger.Info.Println("Webhook server stopped")
	return nil
}

// requestHandler is the main fasthttp request handler
func (ws *WebhookServer) requestHandler(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())

	switch path {
	case ws.webhookPath:
		ws.handleWebhook(ctx)
	case "/health":
		ws.handleHealthCheck(ctx)
	default:
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("Not Found")
	}
}

// handleWebhook processes incoming webhook requests from Telegram
func (ws *WebhookServer) handleWebhook(ctx *fasthttp.RequestCtx) {
	// Check method
	if !ctx.IsPost() {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		ctx.SetBodyString("Method not allowed")
		return
	}

	// Validate secret token from header if configured
	if ws.secretToken != "" {
		receivedToken := string(ctx.Request.Header.Peek("X-Telegram-Bot-Api-Secret-Token"))
		if receivedToken != ws.secretToken {
			logger.Warning.Printf("Invalid secret token received from %s", ctx.RemoteAddr())
			ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			ctx.SetBodyString("Unauthorized")
			return
		}
	}

	// Parse update
	var update telego.Update
	if err := json.Unmarshal(ctx.PostBody(), &update); err != nil {
		logger.Error.Printf("Failed to parse webhook update: %v", err)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString("Bad request")
		return
	}

	// Process update
	go ws.bot.processUpdate(update)

	// Respond with 200 OK
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString("OK")
}

// handleHealthCheck provides a health check endpoint
func (ws *WebhookServer) handleHealthCheck(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString("OK")
}
