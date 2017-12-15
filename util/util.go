package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/prometheus/alertmanager/dispatch"
	"github.com/prometheus/alertmanager/types"
	prommodel "github.com/prometheus/common/model"
	"github.com/zionwu/monitoring-manager/config"
	"github.com/zionwu/monitoring-manager/model"
)

func ReloadConfiguration(url string) error {
	//TODO: what is the wait time
	time.Sleep(10 * time.Second)
	resp, err := http.Post(url+"/-/reload", "text/html", nil)
	logrus.Debugf("Reload  configuration for %s", url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func AddSilence(alert *model.Alert) error {

	url := config.GetConfig().AlertManagerURL

	matchers := []*prommodel.Matcher{}
	m1 := &prommodel.Matcher{
		Name:    "alert_id",
		Value:   alert.Id,
		IsRegex: false,
	}
	matchers = append(matchers, m1)

	m2 := &prommodel.Matcher{
		Name:    "environment",
		Value:   alert.Environment,
		IsRegex: false,
	}
	matchers = append(matchers, m2)

	now := time.Now()
	endsAt := now.AddDate(100, 0, 0)
	silence := prommodel.Silence{
		Matchers:  matchers,
		StartsAt:  now,
		EndsAt:    endsAt,
		CreatedAt: now,
		CreatedBy: "rancherlabs",
		Comment:   "silence",
	}

	silenceData, err := json.Marshal(silence)
	if err != nil {
		return err
	}
	logrus.Debugf(string(silenceData))

	resp, err := http.Post(url+"/api/v1/silences", "application/json", bytes.NewBuffer(silenceData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	logrus.Debugf("add silence: %s", string(res))

	return nil

}

func RemoveSilence(alert *model.Alert) error {

	url := config.GetConfig().AlertManagerURL

	res := struct {
		Data   []*types.Silence `json:"data"`
		Status string           `json:"status"`
	}{}

	req, err := http.NewRequest(http.MethodGet, url+"/api/v1/silences", nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Add("filter", fmt.Sprintf("{%s, %s}", "alert_id="+alert.Id, "environment="+alert.Environment))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	requestBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(requestBytes, &res); err != nil {
		return err
	}

	if res.Status != "success" {
		return fmt.Errorf("Failed to get silence rules for alert")
	}

	for _, s := range res.Data {
		if s.Status.State == types.SilenceStateActive {
			delReq, err := http.NewRequest(http.MethodDelete, url+"/api/v1/silence/"+s.ID, nil)
			if err != nil {
				return err
			}

			delResp, err := client.Do(delReq)
			defer delResp.Body.Close()

			res, err := ioutil.ReadAll(delResp.Body)
			if err != nil {
				return err
			}
			logrus.Debugf("delete silence: %s", string(res))

		}

	}

	return nil
}

func GetState(alert *model.Alert, apiAlerts []*dispatch.APIAlert) (string, *dispatch.APIAlert) {

	for _, a := range apiAlerts {
		if string(a.Labels["alert_id"]) == alert.Id && string(a.Labels["environment"]) == alert.Environment {
			if a.Status.State == types.AlertStateSuppressed {
				return model.AlertStateSuppressed, a
			} else {
				return model.AlertStateActive, a
			}
		}
	}

	return model.AlertStateEnabled, nil

}
