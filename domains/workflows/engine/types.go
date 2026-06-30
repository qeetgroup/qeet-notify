package engine

// StepType identifies what a workflow step does.
type StepType string

const (
	StepTypeChannel   StepType = "channel"   // send via a notification channel
	StepTypeDelay     StepType = "delay"     // pause execution for N seconds
	StepTypeCondition StepType = "condition" // branch based on payload expression
)

// Step is one node in the workflow DAG (stored as JSONB in the DB).
type Step struct {
	ID           string         `json:"id"`
	Type         StepType       `json:"type"`
	Channel      string         `json:"channel,omitempty"`      // email|sms|whatsapp|push|inapp|webhook
	TemplateID   string         `json:"template_id,omitempty"`
	DelaySeconds int            `json:"delay_seconds,omitempty"`
	Condition    string         `json:"condition,omitempty"` // simple JSONB path expression
	TrueStep     string         `json:"true_step,omitempty"`
	FalseStep    string         `json:"false_step,omitempty"`
	NextStep     string         `json:"next_step,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// Event is the payload published to NATS by the API server.
type Event struct {
	TenantID     string         `json:"tenant_id"`
	SubscriberID string         `json:"subscriber_id"`
	Event        string         `json:"event"`
	Payload      map[string]any `json:"payload"`
	WorkflowID   string         `json:"workflow_id,omitempty"`   // filled by engine after lookup
	RunID        string         `json:"workflow_run_id,omitempty"` // filled by engine
}

// ChannelJob is published to a per-channel NATS stream by the workflow engine.
type ChannelJob struct {
	TenantID       string         `json:"tenant_id"`
	NotificationID string         `json:"notification_id"`
	SubscriberID   string         `json:"subscriber_id"`
	Channel        string         `json:"channel"`
	TemplateID     string         `json:"template_id"`
	Payload        map[string]any `json:"payload"`
}
