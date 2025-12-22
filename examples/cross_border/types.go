package main

// TransferRequest represents a money transfer request.
type TransferRequest struct {
	Amount    float64 `json:"amount" mangle:"amount"`
	Recipient string  `json:"recipient" mangle:"recipient"`
	Country   string  `json:"country" mangle:"country"`
}

// TransferResponse represents the outcome of a transfer.
// Includes contextual fields to allow Steering (Routing) based on output analysis.
type TransferResponse struct {
	Status        string `json:"status" mangle:"status"`
	RefCode       string `json:"ref_code" mangle:"ref_code"`
	HasDisclaimer bool   `json:"has_disclaimer" mangle:"has_disclaimer"`
	// Context preserved for steering logic
	Recipient string `json:"recipient" mangle:"recipient"`
	Country   string `json:"country" mangle:"country"`
}
