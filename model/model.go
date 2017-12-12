package model

import (
	"time"

	"github.com/rancher/go-rancher/client"
)

const (
	AlertKind       = "alert"
	AlertConfigKind = "alertConfig"
	RecipientKind   = "recipient"

	AlertStateActive     = "active"
	AlertStateSuppressed = "suppressed"
	AlertStateDisabled   = "disabled"
	AlertStateEnabled    = "enabled"
)

type Error struct {
	client.Resource
	Status   int    `json:"status"`
	Code     string `json:"code"`
	Msg      string `json:"message"`
	Detail   string `json:"detail"`
	BaseType string `json:"baseType"`
}

type AlertConfig struct {
	client.Resource
	ResolveTimeout string          `json:"resolveTimeout"`
	EmailConfig    EmailConfigSpec `json:"emailConfig"`
}

type EmailConfigSpec struct {
	SMTPSmartHost    string `json:"smtpSmartHost"`
	SMTPAuthUserName string `json:"smtpAuthUsername"`
	SMTPAuthPassword string `json:"smtpAuthPassword"`
}

type Alert struct {
	client.Resource
	Description string `json:"description"`
	State       string `json:"state"`
	Severity    string `json:"severity"`
	TargetType  string `json:"targetType"`
	TargetID    string `json:"targetId"`

	HostRule    CommonHealthRule `json:"hostRule"`
	ServiceRule CommonHealthRule `json:"serviceRule"`
	StackRule   CommonHealthRule `json:"stackRule"`

	AdvancedOptions AdvancedOptionsSpec `json:"advancedOptions"`
	MetricRule      MetricRuleSpec      `json:"metricRule"`
	Environment     string              `json:"environment"`
	RecipientID     string              `json:"recipientId"`
	StartsAt        time.Time           `json:"startsAt,omitempty"`
	EndsAt          time.Time           `json:"endsAt,omitempty"`
}

type CommonHealthRule struct {
	HoldDuration string `json:"holdDuration, omitempty"`
}

type MetricRuleSpec struct {
	Expr         string `json:"expr, omitempty"`
	HoldDuration string `json:"holdDuration, omitempty"`
}

type AdvancedOptionsSpec struct {
	InitialWait    string `json:"initialWait, omitempty"`
	RepeatInterval string `json:"repeatInterval, omitempty"`
}

type Recipient struct {
	client.Resource
	Environment   string `json:"environment"`
	RecipientType string `json:"recipientType"`

	EmailRecipient   EmailRecipientSpec   `json:"emailRecipient"`
	WebhookRecipient WebhookRecipientSpec `json:"webhookRecipient"`
}

type WebhookRecipientSpec struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type EmailRecipientSpec struct {
	Address string `json:"address"`
}
