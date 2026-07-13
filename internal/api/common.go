// Package api defines the versioned HTTP wire contract shared by the REST
// server and standalone clients. It deliberately contains no application or
// persistence types.
package api

// Error is a safe, machine-readable API error.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse is the stable envelope for request-level failures.
type ErrorResponse struct {
	Error Error `json:"error"`
}

// BulkSucceeded identifies one successful bulk input.
type BulkSucceeded struct {
	Index int   `json:"index"`
	ID    int64 `json:"id"`
}

// BulkFailed identifies one failed bulk input. ID is absent when the resource
// was not known when the failure occurred.
type BulkFailed struct {
	Index int    `json:"index"`
	ID    *int64 `json:"id,omitempty"`
	Error Error  `json:"error"`
}

// BulkResult accounts for every input in a syntactically valid bulk request.
type BulkResult struct {
	Succeeded []BulkSucceeded `json:"succeeded"`
	Failed    []BulkFailed    `json:"failed"`
}

// IdentitySelector selects exactly one supported external or internal user
// identity.
type IdentitySelector struct {
	UserID      *int64  `json:"userId,omitempty"`
	DiscordID   *string `json:"discordId,omitempty"`
	TelegramID  *string `json:"telegramId,omitempty"`
	PhoneNumber *string `json:"phoneNumber,omitempty"`
	WhatsAppID  *string `json:"whatsappId,omitempty"`
}
