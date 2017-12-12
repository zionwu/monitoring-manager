package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/rancher/go-rancher/api"
	"github.com/zionwu/monitoring-manager/model"
	"github.com/zionwu/monitoring-manager/service"
)

func (s *Server) getAlertConfig(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)

	config, err := service.GetAlertConfig()
	if err != nil {
		return http.StatusNotFound, err
	}

	toAlertConfigResource(apiContext, config)
	if err = apiContext.WriteResource(config); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil

}

func (s *Server) updateAlertConfig(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	requestBytes, err := ioutil.ReadAll(req.Body)
	config := &model.AlertConfig{}

	if err := json.Unmarshal(requestBytes, config); err != nil {
		return http.StatusInternalServerError, err
	}

	config, err = service.CreateOrUpdateAlertConfig(config)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	go func() {
		s.alertChan <- struct{}{}
	}()

	toAlertConfigResource(apiContext, config)
	apiContext.Write(config)
	return http.StatusOK, nil

}
