package sync

type Synchronizer interface {
	Run(stopc <-chan struct{}) error
}

func NewPrometheusTargetSynchronizer() Synchronizer {
	return &prometheusTargetSynchronizer{}
}

func NewAlertStateSynchronizer() Synchronizer {
	return &alertStateSynchronizer{}
}

func NewAlertRouteSynchronizer(alertChan <-chan struct{}) Synchronizer {
	return &alertRouteSynchronizer{alertChan: alertChan}
}

func NewPrometheusRuleSynchronizer(promChan <-chan struct{}) Synchronizer {
	return &prometheusRuleSynchronizer{promChan: promChan}
}
