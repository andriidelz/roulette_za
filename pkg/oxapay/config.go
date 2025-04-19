package oxapay

import (
	"gorm.io/gorm"
)

const (
	oxaPayBaseURL = "https://api.oxapay.com"
	payoutPath    = "/v1/payout"
)

// Config holds the OxaPay configuration
type Config struct {
	APIKey      string
	WebhookKey  string
	CallbackURL string
	DB          *gorm.DB
}
