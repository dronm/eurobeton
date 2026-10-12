
package models

type DistribCommand struct {
	RecipientSessionID string `json:"recipient_session_id"`
	Payload            any    `json:"payload"`
}
