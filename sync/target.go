package sync

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/rancher/go-rancher/v2"
	"github.com/zionwu/monitoring-manager/config"
	"github.com/zionwu/monitoring-manager/util"
	yaml "gopkg.in/yaml.v2"
)

const (
	JobNameCadvisor              = "Cadvisor"
	JobNameRancherHealthExporter = "RancherHealthExporter"
	JobNameNodeExporter          = "NodeExporter"
)

type prometheusTargetSynchronizer struct {
}

func (s *prometheusTargetSynchronizer) Run(stopc <-chan struct{}) error {

	c := config.GetConfig()
	url := fmt.Sprintf("%s/v2-beta/schemas", c.CattleURL)
	rclient, err := client.NewRancherClient(&client.ClientOpts{
		Timeout:   time.Second * 30,
		Url:       url,
		AccessKey: c.CattleAccessKey,
		SecretKey: c.CattleSecretKey,
	})
	if err != nil {
		logrus.Errorf("Error while creating api client: %s", err)
		return nil
	}

	tickChan := time.NewTicker(time.Second * time.Duration(c.SyncIntervalSec)).C

	for {
		select {
		case <-stopc:
			return nil

		case <-tickChan:
			projects, err := rclient.Project.List(&client.ListOpts{})
			if err != nil {
				logrus.Errorf("Error while listing projects: %s", err)
				continue
			}

			promConfig, err := promconfig.LoadFile(c.PrometheusConfig)
			if err != nil {
				logrus.Errorf("Error while loading prometheus config: %s", err)
				continue
			}

			cadvisorConfig := promconfig.ScrapeConfig{JobName: JobNameCadvisor}
			nodeExporterConfig := promconfig.ScrapeConfig{JobName: JobNameNodeExporter}
			rancherExporterConfig := promconfig.ScrapeConfig{JobName: JobNameRancherHealthExporter}

			for _, project := range projects.Data {
				filter := map[string]interface{}{}
				filter["projectId"] = project.Id

				hosts, err := rclient.Host.List(&client.ListOpts{Filters: filter})
				if err != nil {
					logrus.Errorf("Error while listing hosts: %s", err)
					continue
				}

				if len(hosts.Data) == 0 {
					continue
				}

				addTargetToScrapeConfig(&cadvisorConfig, c.CadvisorPort, hosts, project, true)
				addTargetToScrapeConfig(&nodeExporterConfig, c.NodeExporterPort, hosts, project, true)
				addTargetToScrapeConfig(&rancherExporterConfig, c.RancherExporterPort, hosts, project, false)

			}

			scrapeConfigs := []*promconfig.ScrapeConfig{}
			scrapeConfigs = append(scrapeConfigs, &cadvisorConfig, &nodeExporterConfig, &rancherExporterConfig)

			for _, scrapeConfig := range promConfig.ScrapeConfigs {
				if scrapeConfig.JobName != JobNameCadvisor &&
					scrapeConfig.JobName != JobNameNodeExporter &&
					scrapeConfig.JobName != JobNameRancherHealthExporter {
					scrapeConfigs = append(scrapeConfigs, scrapeConfig)
				}
			}

			promConfig.ScrapeConfigs = scrapeConfigs

			configBytes, err := yaml.Marshal(promConfig)
			logrus.Debugf("new generated config: %s", string(configBytes))
			if err != nil {
				logrus.Errorf("Error while marshal the config: %s", err)
				continue
			}

			err = ioutil.WriteFile(c.PrometheusConfig, configBytes, 0777)
			if err != nil {
				logrus.Errorf("Error while writing the config to file: %s", err)
				continue
			}

			util.ReloadConfiguration(c.PrometheusURL)

		}
	}

	return nil
}

func addTargetToScrapeConfig(scrapeConfig *promconfig.ScrapeConfig, port string, hosts *client.HostCollection, project client.Project, global bool) {

	if scrapeConfig.ServiceDiscoveryConfig.StaticConfigs == nil {
		scrapeConfig.ServiceDiscoveryConfig.StaticConfigs = []*promconfig.TargetGroup{}
	}
	tgs := scrapeConfig.ServiceDiscoveryConfig.StaticConfigs

	targets := []model.LabelSet{}

	for _, host := range hosts.Data {
		if global {
			target := model.LabelSet{}
			target[model.AddressLabel] = model.LabelValue(fmt.Sprintf("%s:%s", host.AgentIpAddress, port))
			targets = append(targets, target)
		} else {

			for _, endpoint := range host.PublicEndpoints {
				if strconv.FormatInt(endpoint.Port, 10) == port {
					target := model.LabelSet{}
					target[model.AddressLabel] = model.LabelValue(fmt.Sprintf("%s:%s", host.AgentIpAddress, port))
					targets = append(targets, target)
					break
				}
			}

		}

	}

	labels := model.LabelSet{}
	labels[model.LabelName("environment_id")] = model.LabelValue(project.Id)
	labels[model.LabelName("environment_name")] = model.LabelValue(project.Name)

	targetGroup := promconfig.TargetGroup{
		Labels:  labels,
		Targets: targets,
	}
	tgs = append(tgs, &targetGroup)

	scrapeConfig.ServiceDiscoveryConfig.StaticConfigs = tgs

}
