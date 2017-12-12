package sync

import (
	"io/ioutil"

	"github.com/Sirupsen/logrus"
	prommodel "github.com/prometheus/common/model"
	mconfig "github.com/zionwu/monitoring-manager/config"

	"github.com/zionwu/monitoring-manager/model"

	alertconfig "github.com/zionwu/monitoring-manager/model/alertmanager"

	"github.com/zionwu/monitoring-manager/service"
	"github.com/zionwu/monitoring-manager/util"
	yaml "gopkg.in/yaml.v2"
)

type alertRouteSynchronizer struct {
	alertChan <-chan struct{}
}

func (s *alertRouteSynchronizer) Run(stopc <-chan struct{}) error {

	for {
		select {
		case <-s.alertChan:
			if err := s.sync(); err != nil {
				logrus.Errorf("Error occurred while syncing prometheus rules:  %v", err)
			}

		case <-stopc:
			return nil
		}
	}

}

func (s *alertRouteSynchronizer) sync() error {

	notifier, err := service.GetAlertConfig()
	if err != nil {
		logrus.Errorf("Error while getting notifier: %v", err)
		return err
	}

	alertList, err := service.ListAlert("")
	if err != nil {
		logrus.Errorf("Error while listing alert: %v", err)
		return err
	}

	recipientList, err := service.ListRecipient("")
	if err != nil {
		logrus.Errorf("Error while listing recipient: %v", err)
		return err
	}

	config := getDefaultConfig()

	if notifier.ResolveTimeout != "" {
		rt, err := prommodel.ParseDuration(notifier.ResolveTimeout)
		if err == nil {
			config.Global.ResolveTimeout = rt
		}
	}

	if notifier.EmailConfig.SMTPAuthPassword != "" {
		//config.Global.SMTPAuthIdentity = notifier.EmailConfig.SMTPAuthIdentity
		config.Global.SMTPAuthPassword = alertconfig.Secret(notifier.EmailConfig.SMTPAuthPassword)
		//config.Global.SMTPAuthSecret = alertconfig.Secret(notifier.EmailConfig.SMTPAuthSecret)
		config.Global.SMTPAuthUsername = notifier.EmailConfig.SMTPAuthUserName
		config.Global.SMTPFrom = notifier.EmailConfig.SMTPAuthUserName
		config.Global.SMTPSmarthost = notifier.EmailConfig.SMTPSmartHost
		config.Global.SMTPRequireTLS = false
	}

	for _, recipient := range recipientList {
		s.addReceiver2Config(config, recipient)
	}

	for _, alert := range alertList {
		if alert.State == model.AlertStateDisabled {
			continue
		}
		s.addRoute2Config(config, alert)
	}

	configBytes, err := yaml.Marshal(config)
	logrus.Debugf("after updating: %s", string(configBytes))
	if err != nil {
		return err
	}

	cfg := mconfig.GetConfig()
	err = ioutil.WriteFile(cfg.AlertManagerConfig, configBytes, 0777)
	if err != nil {
		logrus.Errorf("Error while writing the config to file: %s", err)
		return err
	}

	//reload alertmanager
	go util.ReloadConfiguration(cfg.AlertManagerURL)

	return nil
}

func (s *alertRouteSynchronizer) addRoute2Config(config *alertconfig.Config, alert *model.Alert) error {

	envRoutes := &config.Route.Routes
	if envRoutes == nil {
		*envRoutes = []*alertconfig.Route{}
	}

	var envRoute *alertconfig.Route
	for _, r := range *envRoutes {
		if r.Match["environment"] == alert.Environment {
			envRoute = r
			break
		}
	}

	if envRoute == nil {
		match := map[string]string{}
		match["environment"] = alert.Environment
		envRoute = &alertconfig.Route{
			Match:  match,
			Routes: []*alertconfig.Route{},
		}
		*envRoutes = append(*envRoutes, envRoute)
	}

	match := map[string]string{}
	match["alert_id"] = alert.Id
	route := &alertconfig.Route{
		Receiver: alert.RecipientID,
		Match:    match,
	}

	gw, err := prommodel.ParseDuration(alert.AdvancedOptions.InitialWait)
	if err == nil {
		route.GroupWait = &gw
	}
	ri, err := prommodel.ParseDuration(alert.AdvancedOptions.RepeatInterval)
	if err == nil {
		route.RepeatInterval = &ri
	}

	envRoute.Routes = append(envRoute.Routes, route)

	return nil
}

func (s *alertRouteSynchronizer) addReceiver2Config(config *alertconfig.Config, recipient *model.Recipient) error {

	receiver := &alertconfig.Receiver{Name: recipient.Id}
	switch recipient.RecipientType {
	case "webhook":
		webhook := &alertconfig.WebhookConfig{
			URL: recipient.WebhookRecipient.URL,
		}
		receiver.WebhookConfigs = append(receiver.WebhookConfigs, webhook)

	case "email":
		header := map[string]string{}
		header["Subject"] = "Alert from Rancher: {{ (index .Alerts 0).Labels.description}}"
		email := &alertconfig.EmailConfig{
			To:      recipient.EmailRecipient.Address,
			Headers: header,
			//HTML:    "Resource Type:  {{ (index .Alerts 0).Labels.target_type}}\nResource Name:  {{ (index .Alerts 0).Labels.target_id}}\nNamespace:  {{ (index .Alerts 0).Labels.namespace}}\n",
		}
		receiver.EmailConfigs = append(receiver.EmailConfigs, email)
	}

	config.Receivers = append(config.Receivers, receiver)

	return nil
}

func getDefaultConfig() *alertconfig.Config {
	config := alertconfig.Config{}

	resolveTimeout, _ := prommodel.ParseDuration("5m")
	config.Global = &alertconfig.GlobalConfig{
		SlackAPIURL:    "slack_api_url",
		ResolveTimeout: resolveTimeout,
		SMTPRequireTLS: false,
	}

	slackConfigs := []*alertconfig.SlackConfig{}
	initSlackConfig := &alertconfig.SlackConfig{
		Channel: "#alert",
	}
	slackConfigs = append(slackConfigs, initSlackConfig)

	receivers := []*alertconfig.Receiver{}
	initReceiver := &alertconfig.Receiver{
		Name:         "rancherlabs",
		SlackConfigs: slackConfigs,
	}
	receivers = append(receivers, initReceiver)

	config.Receivers = receivers

	groupWait, _ := prommodel.ParseDuration("1m")
	groupInterval, _ := prommodel.ParseDuration("0m")
	repeatInterval, _ := prommodel.ParseDuration("1h")

	config.Route = &alertconfig.Route{
		Receiver:       "rancherlabs",
		GroupWait:      &groupWait,
		GroupInterval:  &groupInterval,
		RepeatInterval: &repeatInterval,
	}

	return &config
}
