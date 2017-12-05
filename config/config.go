package config

import "github.com/urfave/cli"

type Config struct {
	CattleURL           string
	CattleAccessKey     string
	CattleSecretKey     string
	PrometheusURL       string
	PrometheusConfig    string
	CadvisorPort        string
	NodeExporterPort    string
	RancherExporterPort string
	SyncIntervalSec     int
	ListenPort          string
}

var config Config

func Init(c *cli.Context) {
	config.CattleURL = c.String("cattle_url")
	config.CattleAccessKey = c.String("cattle_access_key")
	config.CattleSecretKey = c.String("cattle_secret_key")
	config.PrometheusURL = c.String("prometheus_url")
	config.PrometheusConfig = c.String("prometheus_config")
	config.SyncIntervalSec = c.Int("sync_interval_sec")
	config.CadvisorPort = c.String("cadvisor_port")
	config.NodeExporterPort = c.String("node_exporter_port")
	config.RancherExporterPort = c.String("rancher_exporter_port")
	config.ListenPort = c.String("listen_port")
}

func GetConfig() Config {
	return config
}
