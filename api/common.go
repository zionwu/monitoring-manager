package api

import (
	"net/http"

	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	"github.com/zionwu/monitoring-manager/model"
)

type Server struct {
	promChan  chan<- struct{}
	alertChan chan<- struct{}
}

func NewServer(promChan, alertChan chan<- struct{}) *Server {

	return &Server{
		promChan:  promChan,
		alertChan: alertChan,
	}
}

func newSchema() *client.Schemas {
	schemas := &client.Schemas{}

	schemas.AddType("apiVersion", client.Resource{})
	schemas.AddType("schema", client.Schema{})
	schemas.AddType("error", model.Error{})
	recipientSchema(schemas.AddType("recipient", model.Recipient{}))
	alertSchema(schemas.AddType("alert", model.Alert{}))
	alertConfigSchema(schemas.AddType("config", model.AlertConfig{}))

	return schemas
}

func alertSchema(alert *client.Schema) {

	alert.CollectionMethods = []string{http.MethodGet, http.MethodPost}
	alert.ResourceMethods = []string{http.MethodGet, http.MethodDelete, http.MethodPut}

	severity := alert.ResourceFields["severity"]
	severity.Create = true
	severity.Required = true
	severity.Type = "enum"
	severity.Options = []string{"info", "warning", "critical"}
	severity.Default = "critical"
	alert.ResourceFields["severity"] = severity

	state := alert.ResourceFields["state"]
	state.Create = false
	state.Update = false
	state.Type = "enum"
	state.Default = "enabled"
	state.Options = []string{"enabled", "disabled", "active", "suppressed"}
	alert.ResourceFields["state"] = state

	description := alert.ResourceFields["description"]
	description.Create = true
	description.Update = false
	alert.ResourceFields["description"] = description

	targetType := alert.ResourceFields["targetType"]
	targetType.Create = true
	targetType.Update = false
	targetType.Type = "enum"
	targetType.Options = []string{"host", "service", "stack", "metric"}
	alert.ResourceFields["targetType"] = targetType

	targetId := alert.ResourceFields["targetId"]
	targetId.Create = true
	targetId.Update = false
	alert.ResourceFields["targetId"] = targetId

	recipientId := alert.ResourceFields["recipientId"]
	recipientId.Create = true
	recipientId.Update = true
	recipientId.Type = "reference[recipient]"
	alert.ResourceFields["recipientId"] = recipientId

	environment := alert.ResourceFields["environment"]
	environment.Create = true
	environment.Required = true
	environment.Update = false
	alert.ResourceFields["environment"] = environment

	alert.ResourceActions = map[string]client.Action{
		"silence": {
			Output: "alert",
		},
		"unsilence": {
			Output: "alert",
		},
		"enable": {
			Output: "alert",
		},
		"disable": {
			Output: "alert",
		},
	}
}

func recipientSchema(recipient *client.Schema) {
	recipient.CollectionMethods = []string{http.MethodGet, http.MethodPost}
	recipient.ResourceMethods = []string{http.MethodGet, http.MethodDelete, http.MethodPut}

	environment := recipient.ResourceFields["environment"]
	environment.Create = true
	environment.Required = true
	environment.Update = false
	recipient.ResourceFields["environment"] = environment

	recipientType := recipient.ResourceFields["recipientType"]
	recipientType.Create = true
	recipientType.Update = false
	recipientType.Type = "enum"
	recipientType.Options = []string{"email", "webhook"}
	recipient.ResourceFields["recipientType"] = recipientType
}

func alertConfigSchema(config *client.Schema) {
	config.CollectionMethods = []string{http.MethodGet, http.MethodPost}
	config.ResourceActions = map[string]client.Action{
		"update": client.Action{
			Output: "config",
		},
	}
}

func toAlertConfigResource(apiContext *api.ApiContext, config *model.AlertConfig) *model.AlertConfig {
	config.Resource = client.Resource{
		Type:    "config",
		Actions: map[string]string{},
		Links:   map[string]string{},
	}
	config.Actions["update"] = apiContext.UrlBuilder.Current() + "?action=update"

	return config
}

func toRecipientCollections(apiContext *api.ApiContext, recipients []*model.Recipient) []interface{} {
	var r []interface{}
	for _, p := range recipients {
		r = append(r, toRecipientResource(apiContext, p))
	}
	return r
}

func toRecipientResource(apiContext *api.ApiContext, recipient *model.Recipient) *model.Recipient {
	recipient.Resource = client.Resource{
		Id:      recipient.Id,
		Type:    "recipient",
		Actions: map[string]string{},
		Links:   map[string]string{},
	}

	recipient.Resource.Links["update"] = apiContext.UrlBuilder.ReferenceByIdLink("recipient", recipient.Id)
	recipient.Resource.Links["remove"] = apiContext.UrlBuilder.ReferenceByIdLink("recipient", recipient.Id)
	recipient.Resource.Links["self"] = apiContext.UrlBuilder.ReferenceByIdLink("recipient", recipient.Id)

	return recipient
}

func toAlertCollections(apiContext *api.ApiContext, alerts []*model.Alert) []interface{} {
	var r []interface{}
	for _, p := range alerts {
		r = append(r, toAlertResource(apiContext, p))
	}
	return r
}

func toAlertResource(apiContext *api.ApiContext, alert *model.Alert) *model.Alert {
	alert.Resource = client.Resource{
		Id:      alert.Id,
		Type:    "alert",
		Actions: map[string]string{},
		Links:   map[string]string{},
	}

	alert.Resource.Links["update"] = apiContext.UrlBuilder.ReferenceByIdLink("alert", alert.Id)
	alert.Resource.Links["remove"] = apiContext.UrlBuilder.ReferenceByIdLink("alert", alert.Id)
	alert.Resource.Links["self"] = apiContext.UrlBuilder.ReferenceByIdLink("alert", alert.Id)
	alert.Resource.Links["recipient"] = apiContext.UrlBuilder.ReferenceByIdLink("recipient", alert.RecipientID)
	alert.Actions["enable"] = apiContext.UrlBuilder.ReferenceLink(alert.Resource) + "?action=enable"
	alert.Actions["disable"] = apiContext.UrlBuilder.ReferenceLink(alert.Resource) + "?action=disable"
	alert.Actions["silence"] = apiContext.UrlBuilder.ReferenceLink(alert.Resource) + "?action=silence"
	alert.Actions["unsilence"] = apiContext.UrlBuilder.ReferenceLink(alert.Resource) + "?action=unsilence"

	return alert
}
