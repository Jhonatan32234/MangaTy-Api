package domain

import "time"

type Wallet struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Balance   int       `json:"balance"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Transaction struct {
	ID              string    `json:"id"`
	WalletID        string    `json:"wallet_id"`
	ChapterID       string    `json:"chapter_id"`
	Amount          int       `json:"amount"`
	TransactionType string    `json:"transaction_type"`
	CreatedAt       time.Time `json:"created_at"`
}


type BalanceResponse struct {
	WalletID string `json:"wallet_id"`
	Balance  int    `json:"balance"`
	Currency string `json:"currency"`
}

type UnlockChapterRequest struct {
	ChapterID string `json:"chapter_id" validate:"required,uuid"`
}

type UnlockChapterResponse struct {
	Status        string `json:"status"`
	TransactionID string `json:"transaction_id"`
	NewBalance    int    `json:"new_balance"`
	Message       string `json:"message"`
}