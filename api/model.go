package api

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	v2client "github.com/rancher/go-rancher/v2"
	"github.com/zionwu/monitoring-manager/config"
)

type Error struct {
	client.Resource
	Status   int    `json:"status"`
	Code     string `json:"code"`
	Msg      string `json:"message"`
	Detail   string `json:"detail"`
	BaseType string `json:"baseType"`
}

type Server struct {
	rclient *v2client.RancherClient
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

func newSchema() *client.Schemas {
	schemas := &client.Schemas{}

	schemas.AddType("apiVersion", client.Resource{})
	schemas.AddType("schema", client.Schema{})
	schemas.AddType("error", Error{})
	recipientSchema(schemas.AddType("recipient", Recipient{}))
	alertSchema(schemas.AddType("alert", Alert{}))
	alertConfigSchema(schemas.AddType("config", AlertConfig{}))

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

func NewServer() *Server {

	c := config.GetConfig()
	url := fmt.Sprintf("%s/v2-beta/schemas", c.CattleURL)
	rclient, err := v2client.NewRancherClient(&v2client.ClientOpts{
		Timeout:   time.Second * 30,
		Url:       url,
		AccessKey: c.CattleAccessKey,
		SecretKey: c.CattleSecretKey,
	})
	if err != nil {
		panic(err.Error())
	}

	return &Server{
		rclient: rclient,
	}
}

func toAlertConfigResource(apiContext *api.ApiContext, config *AlertConfig) *AlertConfig {
	config.Resource = client.Resource{
		Type:    "config",
		Actions: map[string]string{},
		Links:   map[string]string{},
	}
	config.Actions["update"] = apiContext.UrlBuilder.Current() + "?action=update"

	return config
}

func (s *Server) paginateGenericObjects(kind string) ([]v2client.GenericObject, error) {
	result := []v2client.GenericObject{}
	limit := "1000"
	marker := ""
	var pageData []v2client.GenericObject
	var err error
	for {
		logrus.Debugf("paging got:%v,%v,%v", kind, limit, marker)
		pageData, marker, err = s.getGenericObjects(kind, limit, marker)
		if err != nil {
			logrus.Debugf("get genericobject err:%v", err)
			return nil, err
		}
		result = append(result, pageData...)
		if marker == "" {
			break
		}
	}
	return result, nil
}

func (s *Server) getGenericObjectById(kind string, id string) (v2client.GenericObject, error) {
	filters := make(map[string]interface{})
	filters["key"] = id
	filters["kind"] = kind
	goCollection, err := s.rclient.GenericObject.List(&v2client.ListOpts{
		Filters: filters,
	})

	if err != nil {
		logrus.Errorf("Error %v filtering genericObjects by key", err)
		return v2client.GenericObject{}, err
	}

	if len(goCollection.Data) == 0 {
		return v2client.GenericObject{}, fmt.Errorf("can not find the recipient for id %s", id)
	}

	return goCollection.Data[0], nil

}

func (s *Server) getGenericObjects(kind string, limit string, marker string) ([]v2client.GenericObject, string, error) {

	filters := make(map[string]interface{})
	filters["kind"] = kind
	filters["limit"] = limit
	filters["marker"] = marker
	goCollection, err := s.rclient.GenericObject.List(&v2client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		logrus.Errorf("fail querying generic objects, error:%v", err)
		return nil, "", err
	}
	//get next marker
	nextMarker := ""
	if goCollection.Pagination != nil && goCollection.Pagination.Next != "" {
		r, err := url.Parse(goCollection.Pagination.Next)
		if err != nil {
			logrus.Errorf("fail parsing next url, error:%v", err)
			return nil, "", err
		}
		nextMarker = r.Query().Get("marker")
	}
	return goCollection.Data, nextMarker, err

}

func toRecipientCollections(apiContext *api.ApiContext, recipients []*Recipient) []interface{} {
	var r []interface{}
	for _, p := range recipients {
		r = append(r, toRecipientResource(apiContext, p))
	}
	return r
}

func toRecipientResource(apiContext *api.ApiContext, recipient *Recipient) *Recipient {
	recipient.Resource = client.Resource{
		Id:      recipient.Id,
		Type:    "recipient",
		Actions: map[string]string{},
		Links:   map[string]string{},
	}

	recipient.Resource.Links["update"] = apiContext.UrlBuilder.ReferenceByIdLink("recipient", recipient.Id)
	recipient.Resource.Links["remove"] = apiContext.UrlBuilder.ReferenceByIdLink("recipient", recipient.Id)

	return recipient
}

func toAlertCollections(apiContext *api.ApiContext, alerts []*Alert) []interface{} {
	var r []interface{}
	for _, p := range alerts {
		r = append(r, toAlertResource(apiContext, p))
	}
	return r
}

func toAlertResource(apiContext *api.ApiContext, alert *Alert) *Alert {
	alert.Resource = client.Resource{
		Id:      alert.Id,
		Type:    "alert",
		Actions: map[string]string{},
		Links:   map[string]string{},
	}

	alert.Resource.Links["update"] = apiContext.UrlBuilder.ReferenceByIdLink("alert", alert.Id)
	alert.Resource.Links["remove"] = apiContext.UrlBuilder.ReferenceByIdLink("alert", alert.Id)

	return alert
}
