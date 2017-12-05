package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/api"
	v2client "github.com/rancher/go-rancher/v2"
	"github.com/sluu99/uuid"
)

func (s *Server) getAlertConfig(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)

	geObjList, err := s.paginateGenericObjects("alertConfig")
	if err != nil {
		logrus.Errorf("fail to list alertConfig,err:%v", err)
		return 0, nil
	}

	if len(geObjList) == 0 {
		//init new settings
		return http.StatusNotFound, errors.New("Could not find alert config")
	}
	data := geObjList[0]
	config := &AlertConfig{}
	if err = json.Unmarshal([]byte(data.ResourceData["data"].(string)), &config); err != nil {
		return http.StatusInternalServerError, err
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
	config := &AlertConfig{}

	if err := json.Unmarshal(requestBytes, config); err != nil {
		return http.StatusInternalServerError, err
	}

	if config.Id == "" {
		config.Id = uuid.Rand().Hex()
	}
	b, err := json.Marshal(config)
	if err != nil {
		return 0, err
	}
	resourceData := map[string]interface{}{
		"data": string(b),
	}

	geObjList, err := s.paginateGenericObjects("alertConfig")
	if err != nil {
		logrus.Errorf("fail to list alertConfig,err:%v", err)
		return 0, nil
	}
	if err != nil {
		logrus.Errorf("Error %v filtering genericObjects by key", err)
		return http.StatusInternalServerError, err
	}
	if len(geObjList) == 0 {
		//not exist,create a setting object
		_, err := s.rclient.GenericObject.Create(&v2client.GenericObject{
			Name:         "alertConfig",
			Key:          "alertConfig",
			ResourceData: resourceData,
			Kind:         "alertConfig",
		})

		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("Save alert config got error: %v", err)
		}

	} else {
		existing := geObjList[0]

		_, err = s.rclient.GenericObject.Update(&existing, &v2client.GenericObject{
			Name:         "alertConfig",
			Key:          "alertConfig",
			ResourceData: resourceData,
			Kind:         "alertConfig",
		})
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("Save alert config got error: %v", err)
		}
	}

	toAlertConfigResource(apiContext, config)
	apiContext.Write(config)
	return http.StatusOK, nil

}
