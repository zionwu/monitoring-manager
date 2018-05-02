package sync

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/rancher/go-rancher/v2"
	"github.com/zionwu/monitoring-manager/config"
	"github.com/zionwu/monitoring-manager/util"
	"gopkg.in/yaml.v2"
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
			// load config
			promConfig, err := promconfig.LoadFile(c.PrometheusConfig)
			if err != nil {
				logrus.Errorf("Error while loading prometheus config: %s", err)
				continue
			}

			expectedScrapes := []*promconfig.ScrapeConfig{
				{JobName: JobNameCadvisor},
				{JobName: JobNameNodeExporter},
				{JobName: JobNameRancherHealthExporter},
			}
			expectedScrapePorts := []string{
				c.CadvisorPort,
				c.NodeExporterPort,
				c.RancherExporterPort,
			}
			actualScrapesUsed := [3]bool{}

			// keep early scrape_configs:
			for _, workedScrape := range promConfig.ScrapeConfigs {
				switch workedScrape.JobName {
				case JobNameCadvisor:
					expectedScrapes[0] = workedScrape
					actualScrapesUsed[0] = true
				case JobNameNodeExporter:
					expectedScrapes[1] = workedScrape
					actualScrapesUsed[1] = true
				case JobNameRancherHealthExporter:
					expectedScrapes[2] = workedScrape
					actualScrapesUsed[2] = true
				}
			}

			// clean up <scrape_config>.static_configs
			for idx, used := range actualScrapesUsed {
				expectedScrapes[idx].ServiceDiscoveryConfig.StaticConfigs = nil

				if !used {
					promConfig.ScrapeConfigs = append(promConfig.ScrapeConfigs, expectedScrapes[idx])
				}
			}

			projects, err := rclient.Project.List(&client.ListOpts{})
			if err != nil {
				logrus.Errorf("Error while listing projects: %s", err)
				continue
			}

			// fill <scrape_config>.static_configs
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

				for idx, scrape := range expectedScrapes {
					var (
						targets      []model.LabelSet
						staticConfig *promconfig.TargetGroup
					)

					// each host will become a scraped endpoint
					for _, host := range hosts.Data {
						targets = append(targets, model.LabelSet{
							model.AddressLabel: model.LabelValue(fmt.Sprintf("%s:%s", host.AgentIpAddress, expectedScrapePorts[idx])),
						})
					}

					staticConfig = &promconfig.TargetGroup{
						Targets: targets,
						Labels: map[model.LabelName]model.LabelValue{
							"environment_id":   model.LabelValue(project.Id),
							"environment_name": model.LabelValue(project.Name),
						},
						Source: project.Id,
					}

					scrape.ServiceDiscoveryConfig.StaticConfigs = append(scrape.ServiceDiscoveryConfig.StaticConfigs, staticConfig)
				}

			}

			// save config
			configBytes, err := yaml.Marshal(promConfig)
			if err != nil {
				logrus.Errorf("Error while marshal the config: %s", err)
				continue
			}
			if logrus.GetLevel() >= logrus.DebugLevel {
				logrus.Debugf("new generated config: %s", string(configBytes))
			}

			err = ioutil.WriteFile(c.PrometheusConfig, configBytes, 0777)
			if err != nil {
				logrus.Errorf("Error while writing the config to file: %s", err)
				continue
			}

			// reload prometheus
			util.ReloadConfiguration(c.PrometheusURL)
		}
	}

	return nil
}

