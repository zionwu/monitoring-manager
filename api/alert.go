package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	v2client "github.com/rancher/go-rancher/v2"
	"github.com/sluu99/uuid"
)

func (s *Server) listAlerts(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)

	geObjList, err := s.paginateGenericObjects("alert")
	if err != nil {
		logrus.Errorf("fail to list alert,err:%v", err)
		return http.StatusInternalServerError, err
	}

	var recipients []*Alert
	for _, gobj := range geObjList {
		b := []byte(gobj.ResourceData["data"].(string))
		a := &Alert{}
		json.Unmarshal(b, a)
		recipients = append(recipients, a)
	}

	apiContext.Write(&client.GenericCollection{
		Data: toAlertCollections(apiContext, recipients),
	})

	return http.StatusOK, nil

}

func (s *Server) createAlert(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	data, err := ioutil.ReadAll(req.Body)
	alert := &Alert{}
	logrus.Debugf("start create alert, get data:%v", string(data))

	if err := json.Unmarshal(data, alert); err != nil {
		return http.StatusInternalServerError, err
	}

	alert.Id = uuid.Rand().Hex()

	b, err := json.Marshal(*alert)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	resourceData := map[string]interface{}{
		"data": string(b),
	}

	_, err = s.rclient.GenericObject.Create(&v2client.GenericObject{
		Name:         alert.Id,
		Key:          alert.Id,
		ResourceData: resourceData,
		Kind:         "alert",
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}
	apiContext.Write(toAlertResource(apiContext, alert))
	return http.StatusOK, nil

}

func (s *Server) getAlertById(id string) (*Alert, error) {
	data, err := s.getGenericObjectById("alert", id)
	if err != nil {
		return nil, err
	}

	alert := &Alert{}
	err = json.Unmarshal([]byte(data.ResourceData["data"].(string)), alert)
	if err != nil {
		return nil, err
	}

	return alert, nil
}

func (s *Server) getAlert(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	alert, err := s.getAlertById(id)
	if err != nil {
		logrus.Errorf("Error while getting alert", err)
		return http.StatusInternalServerError, err
	}

	apiContext.Write(toAlertResource(apiContext, alert))

	return http.StatusOK, nil
}

func (s *Server) deleteAlert(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	data, err := s.getGenericObjectById("alert", id)
	if err != nil {
		logrus.Errorf("Error while getting alert", err)
		return http.StatusInternalServerError, err
	}

	if err = s.rclient.GenericObject.Delete(&data); err != nil {
		return http.StatusInternalServerError, err
	}

	alert := &Alert{}
	err = json.Unmarshal([]byte(data.ResourceData["data"].(string)), alert)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	apiContext.Write(toAlertResource(apiContext, alert))
	return http.StatusOK, nil

}

func (s *Server) updateAlert(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	r, err := s.getGenericObjectById("alert", id)
	if err != nil {
		logrus.Errorf("Error while getting alert", err)
		return http.StatusInternalServerError, err
	}

	alert := &Alert{}
	data, err := ioutil.ReadAll(req.Body)
	if err := json.Unmarshal(data, alert); err != nil {
		return http.StatusInternalServerError, err
	}

	b, err := json.Marshal(*alert)
	if err != nil {
		return 0, err
	}
	resourceData := map[string]interface{}{
		"data": string(b),
	}

	_, err = s.rclient.GenericObject.Update(&r, &v2client.GenericObject{
		Name:         id,
		Key:          id,
		ResourceData: resourceData,
		Kind:         "alert",
	})

	apiContext.Write(toAlertResource(apiContext, alert))
	return http.StatusOK, nil
}
