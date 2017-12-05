package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/urfave/cli"
	"github.com/zionwu/monitoring-manager/api"
	"github.com/zionwu/monitoring-manager/config"
	"github.com/zionwu/monitoring-manager/sync"
	"golang.org/x/sync/errgroup"
)

var VERSION = "v0.0.1"

func main() {

	app := cli.NewApp()
	app.Name = "monitoring-manager"
	app.Version = VERSION
	app.Usage = "A monitoring manager for Rancher"
	app.Action = run
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "debug, d",
			Usage:  "Debug logging",
			EnvVar: "DEBUG",
		},
		cli.StringFlag{
			Name:   "prometheus_url, p",
			Usage:  "Prometheus URL",
			EnvVar: "PROMETHEUS_URL",
			Value:  "http://prometheus:9090",
		},
		cli.StringFlag{
			Name:   "prometheus_config",
			Usage:  "Prometheus Config",
			EnvVar: "PROMETHEUS_CONFIG",
			Value:  "/etc/prometheus/prometheus.yml",
		},
		cli.IntFlag{
			Name:   "sync_interval_sec, i",
			Usage:  "prometheus target synchronize interval",
			EnvVar: "SYNC_INTERVAL_SEC",
			Value:  30,
		},
		cli.StringFlag{
			Name:   "cattle_url",
			Usage:  "cattle url",
			EnvVar: "CATTLE_URL",
		},
		cli.StringFlag{
			Name:   "cattle_access_key",
			Usage:  "cattle access key",
			EnvVar: "CATTLE_ACCESS_KEY",
		},

		cli.StringFlag{
			Name:   "cattle_secret_key",
			Usage:  "cattle secret key",
			EnvVar: "CATTLE_SECRET_KEY",
		},
		cli.StringFlag{
			Name:   "cadvisor_port",
			Usage:  "cadvisor port",
			EnvVar: "CADVISOR_PORT",
			Value:  "9101",
		},
		cli.StringFlag{
			Name:   "node_exporter_port",
			Usage:  "node exporter port",
			EnvVar: "NODE_EXPORTER_PORT",
			Value:  "9100",
		},
		cli.StringFlag{
			Name:   "rancher_exporter_port",
			Usage:  "rancher exporter port",
			EnvVar: "RANCHER_EXPORTRT_PORT",
			Value:  "9173",
		},
		cli.StringFlag{
			Name:   "listen_port, l",
			Usage:  "listen port",
			EnvVar: "LISTEN_PORT",
			Value:  "8888",
		},
	}

	app.Run(os.Args)
}

func run(c *cli.Context) error {
	if c.Bool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	}

	config.Init(c)

	router := http.Handler(api.NewRouter(api.NewServer()))
	router = handlers.LoggingHandler(os.Stdout, router)
	router = handlers.ProxyHeaders(router)
	logrus.Infof("Alertmanager operator running on %s", config.GetConfig().ListenPort)
	go http.ListenAndServe(":"+config.GetConfig().ListenPort, router)

	ctx, cancel := context.WithCancel(context.Background())
	wg, ctx := errgroup.WithContext(ctx)

	wg.Go(func() error { return sync.NewPrometheusTargetSynchronizer().Run(ctx.Done()) })

	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	select {
	case <-term:
		logrus.Info("msg", "Received SIGTERM, exiting gracefully...")
	case <-ctx.Done():
	}

	cancel()
	if err := wg.Wait(); err != nil {
		logrus.Errorf("msg", "Unhandled error received. Exiting: %v", err)
		return err
	}

	return nil
}
