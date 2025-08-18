package audit

import "time"

// AuditEvent defines the structure for a detailed, structured audit log entry.
type AuditEvent struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	EventType     string                 `json:"event_type"`
	Actor         *ActorInfo             `json:"actor"`
	Action        string                 `json:"action"`
	Resource      *ResourceInfo          `json:"resource"`
	Result        string                 `json:"result"`
	Details       map[string]interface{} `json:"details"`
	SecurityLevel string                 `json:"security_level"`
	Checksum      string                 `json:"checksum"`
}

// ActorInfo captures information about the entity performing the action.
type ActorInfo struct {
	UserID      string `json:"user_id"`
	ClientIP    string `json:"client_ip"`
	UserAgent   string `json:"user_agent"`
	SessionID   string `json:"session_id"`
}

// ResourceInfo describes the resource that was acted upon.
type ResourceInfo struct {
	Type            string `json:"type"`
	ID              string `json:"id"`
	Classification  string `json:"classification"`
	Attributes      map[string]string `json:"attributes"`
}

// KeyOperationRequest is a struct to pass all necessary information for auditing a key operation.
type KeyOperationRequest struct {
	Operation          string
	KeyID              string
	DataClassification string
	Result             string
	SessionID          string
	ClientIP           string
	UserAgent          string
	AdditionalContext  map[string]interface{}
}
