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

func (s *Server) listRecipient(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)

	geObjList, err := s.paginateGenericObjects("recipient")
	if err != nil {
		logrus.Errorf("fail to list alertConfig,err:%v", err)
		return http.StatusInternalServerError, err
	}

	var recipients []*Recipient
	for _, gobj := range geObjList {
		b := []byte(gobj.ResourceData["data"].(string))
		a := &Recipient{}
		json.Unmarshal(b, a)
		recipients = append(recipients, a)
	}

	apiContext.Write(&client.GenericCollection{
		Data: toRecipientCollections(apiContext, recipients),
	})

	return http.StatusOK, nil

}

func (s *Server) createRecipient(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	data, err := ioutil.ReadAll(req.Body)
	recipient := &Recipient{}
	logrus.Debugf("start create recipient, get data:%v", string(data))

	if err := json.Unmarshal(data, recipient); err != nil {
		return http.StatusInternalServerError, err
	}

	recipient.Id = uuid.Rand().Hex()

	b, err := json.Marshal(*recipient)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	resourceData := map[string]interface{}{
		"data": string(b),
	}

	_, err = s.rclient.GenericObject.Create(&v2client.GenericObject{
		Name:         recipient.Id,
		Key:          recipient.Id,
		ResourceData: resourceData,
		Kind:         "recipient",
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}
	apiContext.Write(toRecipientResource(apiContext, recipient))
	return http.StatusOK, nil

}

func (s *Server) getRecipientById(id string) (*Recipient, error) {
	data, err := s.getGenericObjectById("recipient", id)
	if err != nil {
		return nil, err
	}

	recipient := &Recipient{}
	err = json.Unmarshal([]byte(data.ResourceData["data"].(string)), recipient)
	if err != nil {
		return nil, err
	}

	return recipient, nil
}

func (s *Server) getRecipient(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	recipient, err := s.getRecipientById(id)
	if err != nil {
		logrus.Errorf("Error while getting recipient", err)
		return http.StatusInternalServerError, err
	}

	apiContext.Write(toRecipientResource(apiContext, recipient))

	return http.StatusOK, nil
}

func (s *Server) deleteRecipient(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	data, err := s.getGenericObjectById("recipient", id)
	if err != nil {
		logrus.Errorf("Error while getting recipient", err)
		return http.StatusInternalServerError, err
	}

	if err = s.rclient.GenericObject.Delete(&data); err != nil {
		return http.StatusInternalServerError, err
	}

	recipient := &Recipient{}
	err = json.Unmarshal([]byte(data.ResourceData["data"].(string)), recipient)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	apiContext.Write(toRecipientResource(apiContext, recipient))
	return http.StatusOK, nil

}

func (s *Server) updateRecipient(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	r, err := s.getGenericObjectById("recipient", id)
	if err != nil {
		logrus.Errorf("Error while getting recipient", err)
		return http.StatusInternalServerError, err
	}

	recipient := &Recipient{}
	data, err := ioutil.ReadAll(req.Body)
	if err := json.Unmarshal(data, recipient); err != nil {
		return http.StatusInternalServerError, err
	}

	b, err := json.Marshal(*recipient)
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
		Kind:         "recipient",
	})

	apiContext.Write(toRecipientResource(apiContext, recipient))
	return http.StatusOK, nil
}
