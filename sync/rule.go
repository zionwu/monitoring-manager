package sync

import (
	"io/ioutil"

	"github.com/Sirupsen/logrus"
	prommodel "github.com/prometheus/common/model"
	"github.com/zionwu/monitoring-manager/config"
	"github.com/zionwu/monitoring-manager/model"
	"github.com/zionwu/monitoring-manager/service"

	"github.com/zionwu/monitoring-manager/util"
	yaml "gopkg.in/yaml.v2"
)

type prometheusRuleSynchronizer struct {
	promChan <-chan struct{}
}

func (s *prometheusRuleSynchronizer) Run(stopc <-chan struct{}) error {

	for {
		select {
		case <-s.promChan:
			if err := s.sync(); err != nil {
				logrus.Errorf("Error occurred while syncing prometheus rules:  %v", err)
			}

		case <-stopc:
			return nil
		}
	}

}

func (s *prometheusRuleSynchronizer) sync() error {

	//fs := fields.OneTermEqualSelector("targetType", "metric").String()
	alertList, err := service.ListAlert("")
	if err != nil {
		logrus.Errorf("Error while listing alert: %v", err)
		return err
	}

	rules := []Rule{}

	for _, alert := range alertList {

		if alert.State == model.AlertStateDisabled {
			continue
		}

		labels := map[string]string{}
		labels["alert_id"] = alert.Id
		labels["severity"] = alert.Severity
		labels["description"] = alert.Description
		labels["target_type"] = alert.TargetType
		labels["environment"] = alert.Environment

		switch alert.TargetType {
		case "metric":
			holdDuration, _ := prommodel.ParseDuration(alert.MetricRule.HoldDuration)

			rule := Rule{
				Alert:  alert.Description,
				Expr:   alert.MetricRule.Expr,
				For:    holdDuration,
				Labels: labels,
			}
			rules = append(rules, rule)
		case "service":
			holdDuration, _ := prommodel.ParseDuration(alert.ServiceRule.HoldDuration)
			expr := "rancher_service_health_status{environment_id=\"" + alert.Environment + "\", id=\"" + alert.TargetID + "\", health_state=\"healthy\"} != 1"

			rule := Rule{
				Alert:  alert.Description,
				Expr:   expr,
				For:    holdDuration,
				Labels: labels,
			}
			rules = append(rules, rule)

		case "stack":
			holdDuration, _ := prommodel.ParseDuration(alert.StackRule.HoldDuration)
			expr := "rancher_stack_health_status{environment_id=\"" + alert.Environment + "\", id=\"" + alert.TargetID + "\", health_state=\"healthy\"} != 1"

			rule := Rule{
				Alert:  alert.Description,
				Expr:   expr,
				For:    holdDuration,
				Labels: labels,
			}
			rules = append(rules, rule)

		case "host":
			holdDuration, _ := prommodel.ParseDuration(alert.HostRule.HoldDuration)
			expr := "rancher_host_agent_state{environment_id=\"" + alert.Environment + "\", id=\"" + alert.TargetID + "\", state=\"active\"} != 1"

			rule := Rule{
				Alert:  alert.Description,
				Expr:   expr,
				For:    holdDuration,
				Labels: labels,
			}
			rules = append(rules, rule)
		}

	}

	rg := RuleGroup{
		Name:  "rancher-rules",
		Rules: rules,
	}
	ruleGroups := []RuleGroup{}
	ruleGroups = append(ruleGroups, rg)

	rgs := RuleGroups{
		Groups: ruleGroups,
	}

	ruleStr, err := yaml.Marshal(rgs)
	logrus.Debugf("after updating rules: %s", string(ruleStr))
	if err != nil {
		return err
	}

	c := config.GetConfig()
	err = ioutil.WriteFile(c.PrometheusRule, ruleStr, 0777)
	if err != nil {
		logrus.Errorf("Error while writing the config to file: %s", err)
		return err
	}

	//reload prometheus configuration
	go util.ReloadConfiguration(c.PrometheusURL)

	return nil
}

// RuleGroups is a set of rule groups that are typically exposed in a file.
type RuleGroups struct {
	Groups []RuleGroup `yaml:"groups"`
}

// RuleGroup is a list of sequentially evaluated recording and alerting rules.
type RuleGroup struct {
	Name     string             `yaml:"name"`
	Interval prommodel.Duration `yaml:"interval,omitempty"`
	Rules    []Rule             `yaml:"rules"`
}

// Rule describes an alerting or recording rule.
type Rule struct {
	Record      string             `yaml:"record,omitempty"`
	Alert       string             `yaml:"alert,omitempty"`
	Expr        string             `yaml:"expr"`
	For         prommodel.Duration `yaml:"for,omitempty"`
	Labels      map[string]string  `yaml:"labels,omitempty"`
	Annotations map[string]string  `yaml:"annotations,omitempty"`
}
