package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	"github.com/zionwu/monitoring-manager/model"
	"github.com/zionwu/monitoring-manager/service"
	"github.com/zionwu/monitoring-manager/util"
)

func (s *Server) listAlerts(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)

	var environment string
	vals := req.URL.Query()
	if nsarr, ok := vals["environment"]; ok {
		environment = nsarr[0]
	}

	alerts, err := service.ListAlert(environment)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	apiContext.Write(&client.GenericCollection{
		Data: toAlertCollections(apiContext, alerts),
	})

	return http.StatusOK, nil
}

func (s *Server) createAlert(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	data, err := ioutil.ReadAll(req.Body)
	logrus.Debugf("start create alert, get data:%v", string(data))

	alert := &model.Alert{State: model.AlertStateEnabled}
	if err := json.Unmarshal(data, alert); err != nil {
		return http.StatusInternalServerError, err
	}

	if err = s.checkAlertParam(alert); err != nil {
		return http.StatusBadRequest, err
	}

	_, err = service.GetRecipient(alert.RecipientID)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("unable to find the recipient: %v", err)
	}

	err = service.CreateAlert(alert)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	go func() {
		s.alertChan <- struct{}{}
	}()

	go func() {
		s.promChan <- struct{}{}
	}()

	apiContext.Write(toAlertResource(apiContext, alert))
	return http.StatusOK, nil

}

func (s *Server) getAlert(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	alert, err := service.GetAlert(id)
	if err != nil {
		return http.StatusNotFound, err
	}

	apiContext.Write(toAlertResource(apiContext, alert))

	return http.StatusOK, nil
}

func (s *Server) deleteAlert(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	alert, err := service.GetAlert(id)
	if err != nil {
		logrus.Errorf("Error while getting alert", err)
		return http.StatusBadRequest, err
	}

	err = service.DeleteAlert(id)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	apiContext.Write(toAlertResource(apiContext, alert))
	return http.StatusOK, nil

}

func (s *Server) updateAlert(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	alert := &model.Alert{}
	data, err := ioutil.ReadAll(req.Body)
	if err := json.Unmarshal(data, alert); err != nil {
		return http.StatusInternalServerError, err
	}
	alert.Id = id

	oriAlert, err := service.GetAlert(id)
	if err != nil {
		return http.StatusNotFound, err
	}

	if err = s.checkAlertParam(alert); err != nil {
		return http.StatusBadRequest, err
	}

	_, err = service.GetRecipient(alert.RecipientID)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("unable to find the recipient: %v", err)
	}

	alert.State = oriAlert.State
	err = service.UpdateAlert(alert)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	go func() {
		s.alertChan <- struct{}{}
	}()

	go func() {
		s.promChan <- struct{}{}
	}()

	apiContext.Write(toAlertResource(apiContext, alert))
	return http.StatusOK, nil
}

func (s *Server) deactivateAlert(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	alert, err := service.GetAlert(id)
	if err != nil {
		return http.StatusNotFound, err
	}

	if alert.State != model.AlertStateEnabled {
		return http.StatusBadRequest, fmt.Errorf("Current state is not enabled, can not perform disable action")
	}

	alert.State = model.AlertStateDisabled
	err = service.UpdateAlert(alert)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	go func() {
		s.alertChan <- struct{}{}
	}()

	go func() {
		s.promChan <- struct{}{}
	}()

	apiContext.Write(toAlertResource(apiContext, alert))
	return http.StatusOK, nil
}

func (s *Server) activateAlert(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	alert, err := service.GetAlert(id)
	if err != nil {
		return http.StatusNotFound, err
	}

	if alert.State != model.AlertStateDisabled {
		return http.StatusBadRequest, fmt.Errorf("Current state is not disabled, can not perform enable action")
	}

	alert.State = model.AlertStateEnabled
	err = service.UpdateAlert(alert)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	go func() {
		s.alertChan <- struct{}{}
	}()

	go func() {
		s.promChan <- struct{}{}
	}()

	apiContext.Write(toAlertResource(apiContext, alert))
	return http.StatusOK, nil
}

func (s *Server) silenceAlert(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {

	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	alert, err := service.GetAlert(id)
	if err != nil {
		return http.StatusNotFound, err
	}

	if alert.State != model.AlertStateActive {
		return http.StatusBadRequest, fmt.Errorf("Current state is not active, can not perform slience action")
	}

	err = util.AddSilence(alert)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("Error while adding silence to AlertManager: %v", err)
	}

	alert.State = model.AlertStateSuppressed
	err = service.UpdateAlert(alert)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	apiContext.Write(toAlertResource(apiContext, alert))
	return http.StatusOK, nil

}

func (s *Server) unsilenceAlert(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {

	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	alert, err := service.GetAlert(id)
	if err != nil {
		return http.StatusNotFound, err
	}

	if alert.State != model.AlertStateSuppressed {
		return http.StatusBadRequest, fmt.Errorf("Current state is not active, can not perform slience action")
	}

	err = util.RemoveSilence(alert)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("Error while adding silence to AlertManager: %v", err)
	}

	alert.State = model.AlertStateActive
	err = service.UpdateAlert(alert)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	apiContext.Write(toAlertResource(apiContext, alert))
	return http.StatusOK, nil

}

func (s *Server) checkAlertParam(alert *model.Alert) error {

	if alert.Environment == "" {
		return fmt.Errorf("missing environment")
	}

	if alert.Description == "" {
		return fmt.Errorf("missing description")
	}

	if alert.RecipientID == "" {
		return fmt.Errorf("missing Recipient ID")
	}

	if !(alert.TargetType == "host" || alert.TargetType == "stack" || alert.TargetType == "service" || alert.TargetType == "metric") {
		return fmt.Errorf("Invalid Target Type")
	}

	if alert.TargetType != "metric" && alert.TargetID == "" {
		return fmt.Errorf("missing Target Id")
	}

	return nil
}
