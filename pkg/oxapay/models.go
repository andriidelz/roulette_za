package oxapay

import (
	"time"
)

// Payout represents a payout request to OxaPay
type Payout struct {
	ID              string  `json:"id" gorm:"primaryKey"`
	Currency        string  `json:"currency"`
	Amount          float64 `json:"amount"`
	Address         string  `json:"address"`
	Description     string  `json:"description,omitempty"`
	UserID          string  `json:"userId,omitempty" gorm:"index"`
	Status          string  `json:"status" gorm:"index"`
	TransactionHash string  `json:"transactionHash,omitempty"`
	Network         string  `json:"network,omitempty"`
	Fee             float64 `json:"fee,omitempty"`
	Memo            string  `json:"memo,omitempty"`
	Internal        bool    `json:"internal,omitempty"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PayoutRequest represents the request to OxaPay API
type PayoutRequest struct {
	Currency    string  `json:"currency"`
	Amount      float64 `json:"amount"`
	Address     string  `json:"address"`
	Description string  `json:"description,omitempty"`
	UserID      string  `json:"user_id,omitempty"`
	Network     string  `json:"network,omitempty"`
	Memo        string  `json:"memo,omitempty"`
}

// PayoutResponse represents the response from OxaPay API
type PayoutResponse struct {
	Data struct {
		TrackID string `json:"track_id"`
		Status  string `json:"status"`
	} `json:"data"`
	Message string      `json:"message"`
	Error   interface{} `json:"error"`
	Status  int         `json:"status"`
	Version string      `json:"version"`
}

// PaginationMeta represents pagination metadata for list responses
type PaginationMeta struct {
	Page     int `json:"page"`
	LastPage int `json:"last_page"`
	Total    int `json:"total"`
}

// PayoutStatusTypes определяет возможные статусы выплат
const (
	PayoutStatusProcessing = "processing" // Request submitted and is being processed
	PayoutStatusPending    = "pending"    // Request processed and is now in the payment queue
	PayoutStatusConfirming = "confirming" // Transaction created and is awaiting confirmation on the blockchain
	PayoutStatusConfirmed  = "confirmed"  // Transaction paid successfully
	PayoutStatusCanceled   = "canceled"   // Payout request was canceled
	PayoutStatusRejected   = "rejected"   // The request was rejected due to some reasons
)
