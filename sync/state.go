package sync

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/prometheus/alertmanager/dispatch"
	"github.com/zionwu/monitoring-manager/config"
	"github.com/zionwu/monitoring-manager/model"
	"github.com/zionwu/monitoring-manager/service"

	"github.com/zionwu/monitoring-manager/util"
)

type alertStateSynchronizer struct {
}

func (s *alertStateSynchronizer) Run(stopc <-chan struct{}) error {

	tickChan := time.NewTicker(time.Second * 30).C

	for {
		select {
		case <-tickChan:
			apiAlerts, err := getActiveAlertListFromAlertManager()
			if err != nil {
				logrus.Errorf("Error while getting alert list from alertmanager: %v", err)
			} else {

				al, err := service.ListAlert("")

				if err != nil {
					logrus.Errorf("Error while geting alert CRD list: %v", err)
				} else {

					for _, alert := range al {
						if alert.State == model.AlertStateDisabled {
							continue
						}

						state, a := util.GetState(alert, apiAlerts)
						needUpdate := false

						//only take ation when the state is not the same
						if state != alert.State {

							//if the origin state is silenced, and current state is active, then need to remove the silence rule
							if alert.State == model.AlertStateSuppressed && state == model.AlertStateEnabled {
								util.RemoveSilence(alert)
							}

							alert.State = state
							needUpdate = true
						}

						if state == model.AlertStateSuppressed || state == model.AlertStateActive {
							if !alert.StartsAt.Equal(a.StartsAt) {
								alert.StartsAt = a.StartsAt
								needUpdate = true
							}

							if !alert.EndsAt.Equal(a.EndsAt) {
								alert.EndsAt = a.EndsAt
								needUpdate = true
							}
						} else {
							alert.StartsAt = time.Time{}
							alert.EndsAt = time.Time{}
						}

						if needUpdate {

							err := service.UpdateAlert(alert)

							if err != nil {
								logrus.Errorf("Error occurred while syn alert state and time: %v", err)
							}
						}

					}
				}
			}

		case <-stopc:
			return nil
		}
	}

}

func getActiveAlertListFromAlertManager() ([]*dispatch.APIAlert, error) {

	url := config.GetConfig().AlertManagerURL
	res := struct {
		Data   []*dispatch.APIAlert `json:"data"`
		Status string               `json:"status"`
	}{}

	req, err := http.NewRequest(http.MethodGet, url+"/api/v1/alerts", nil)
	if err != nil {
		return nil, err
	}
	//q := req.URL.Query()
	//q.Add("filter", fmt.Sprintf("{%s}", filter))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	requestBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(requestBytes, &res); err != nil {
		return nil, err
	}

	return res.Data, nil
}
