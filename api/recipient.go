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
)

func (s *Server) listRecipient(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)

	var environment string
	vals := req.URL.Query()
	if nsarr, ok := vals["environment"]; ok {
		environment = nsarr[0]
	}

	recipients, err := service.ListRecipient(environment)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	apiContext.Write(&client.GenericCollection{
		Data: toRecipientCollections(apiContext, recipients),
	})

	return http.StatusOK, nil

}

func (s *Server) createRecipient(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)

	data, err := ioutil.ReadAll(req.Body)
	recipient := &model.Recipient{}
	logrus.Debugf("start create recipient, get data:%v", string(data))
	if err := json.Unmarshal(data, recipient); err != nil {
		return http.StatusInternalServerError, err
	}

	if err = s.checkRecipientParam(recipient); err != nil {
		return http.StatusBadRequest, err
	}

	err = service.CreateRecipient(recipient)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	go func() {
		s.alertChan <- struct{}{}
	}()

	apiContext.Write(toRecipientResource(apiContext, recipient))
	return http.StatusOK, nil

}

func (s *Server) getRecipient(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	recipient, err := service.GetRecipient(id)
	if err != nil {
		logrus.Errorf("Error while getting recipient", err)
		return http.StatusNotFound, err
	}

	apiContext.Write(toRecipientResource(apiContext, recipient))

	return http.StatusOK, nil
}

func (s *Server) deleteRecipient(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	recipient, err := service.GetRecipient(id)
	if err != nil {
		return http.StatusNotFound, err
	}

	//check if the recipient is used by any alert
	alertList, err := service.ListAlert("")
	if err != nil {
		return http.StatusInternalServerError, err
	}
	for _, alert := range alertList {
		if alert.RecipientID == recipient.Id {
			return http.StatusBadRequest, fmt.Errorf("The recipient %s is still used for alert", id)
		}
	}

	err = service.DeleteRecipient(id)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	go func() {
		s.alertChan <- struct{}{}
	}()

	apiContext.Write(toRecipientResource(apiContext, recipient))
	return http.StatusOK, nil

}

func (s *Server) updateRecipient(rw http.ResponseWriter, req *http.Request) (errCode int, err error) {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	recipient := &model.Recipient{}
	data, err := ioutil.ReadAll(req.Body)
	if err := json.Unmarshal(data, recipient); err != nil {
		return http.StatusInternalServerError, err
	}

	_, err = service.GetRecipient(id)
	if err != nil {
		return http.StatusNotFound, err
	}

	if err = s.checkRecipientParam(recipient); err != nil {
		return http.StatusBadRequest, err
	}
	recipient.Id = id

	service.UpdateRecipient(recipient)

	go func() {
		s.alertChan <- struct{}{}
	}()

	apiContext.Write(toRecipientResource(apiContext, recipient))
	return http.StatusOK, nil
}

func (s *Server) checkRecipientParam(recipient *model.Recipient) error {

	recipientType := recipient.RecipientType
	if !(recipientType == "email" || recipientType == "webhook") {
		return fmt.Errorf("recipientTpye should be email/webhook")
	}

	if recipient.Environment == "" {
		return fmt.Errorf("missing environment")
	}

	switch recipientType {
	case "email":
		if recipient.EmailRecipient.Address == "" {
			return fmt.Errorf("email address can't be empty")
		}
	case "webhook":
		if recipient.WebhookRecipient.URL == "" {
			return fmt.Errorf("webhook url can't be empty")
		}

		if recipient.WebhookRecipient.Name == "" {
			return fmt.Errorf("webhook name can't be empty")
		}
	}

	return nil
}
