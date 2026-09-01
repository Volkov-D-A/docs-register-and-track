package models

import "time"

// TechnicalLogEvent is the bounded transport contract used by desktop to
// submit technical diagnostics to docflow-server. The server owns identity and
// final delivery to Seq.
type TechnicalLogEvent struct {
	Timestamp  time.Time         `json:"timestamp"`
	Level      string            `json:"level"`
	Message    string            `json:"message"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type TechnicalLogBatch struct {
	Events []TechnicalLogEvent `json:"events"`
}
