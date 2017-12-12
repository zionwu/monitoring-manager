package service

import (
	"encoding/json"

	"github.com/Sirupsen/logrus"
	"github.com/sluu99/uuid"
	"github.com/zionwu/monitoring-manager/model"

	v2client "github.com/rancher/go-rancher/v2"
)

func ListRecipient(environment string) ([]*model.Recipient, error) {
	geObjList, err := paginateGenericObjects("recipient")
	if err != nil {
		logrus.Errorf("fail to list alertConfig,err:%v", err)
		return nil, err
	}

	var recipients []*model.Recipient
	for _, gobj := range geObjList {
		b := []byte(gobj.ResourceData["data"].(string))
		a := &model.Recipient{}
		json.Unmarshal(b, a)
		if environment == "" || a.Environment == environment {
			recipients = append(recipients, a)
		}
	}

	return recipients, nil
}

func DeleteRecipient(id string) error {

	rclient, err := getRancherClient()
	if err != nil {
		return err
	}

	data, err := getGenericObjectById("recipient", id)
	if err != nil {
		return err
	}

	if err = rclient.GenericObject.Delete(&data); err != nil {
		return err
	}

	return nil
}

func GetRecipient(id string) (*model.Recipient, error) {
	data, err := getGenericObjectById("recipient", id)
	if err != nil {
		return nil, err
	}

	recipient := &model.Recipient{}
	err = json.Unmarshal([]byte(data.ResourceData["data"].(string)), recipient)
	if err != nil {
		return nil, err
	}

	return recipient, nil
}

func CreateRecipient(recipient *model.Recipient) error {

	rclient, err := getRancherClient()
	if err != nil {
		return err
	}

	recipient.Id = uuid.Rand().Hex()

	b, err := json.Marshal(*recipient)
	if err != nil {
		return err
	}
	resourceData := map[string]interface{}{
		"data": string(b),
	}

	_, err = rclient.GenericObject.Create(&v2client.GenericObject{
		Name:         recipient.Id,
		Key:          recipient.Id,
		ResourceData: resourceData,
		Kind:         "recipient",
	})
	if err != nil {
		return err
	}

	return nil
}

func UpdateRecipient(recipient *model.Recipient) error {

	rclient, err := getRancherClient()
	if err != nil {
		return err
	}

	recipientGO, err := getGenericObjectById("recipient", recipient.Id)
	if err != nil {
		return err
	}

	b, err := json.Marshal(*recipient)
	if err != nil {
		return err
	}
	resourceData := map[string]interface{}{
		"data": string(b),
	}

	_, err = rclient.GenericObject.Update(&recipientGO, &v2client.GenericObject{
		Name:         recipient.Id,
		Key:          recipient.Id,
		ResourceData: resourceData,
		Kind:         "recipient",
	})
	return nil
}
