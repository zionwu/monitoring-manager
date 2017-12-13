package service

import (
	"encoding/json"
	"fmt"

	"github.com/Sirupsen/logrus"
	v2client "github.com/rancher/go-rancher/v2"
	"github.com/sluu99/uuid"
	"github.com/zionwu/monitoring-manager/model"
)

func ListAlert(environment string) ([]*model.Alert, error) {
	geObjList, err := paginateGenericObjects("alert")
	if err != nil {
		logrus.Errorf("fail to list alert,err:%v", err)
		return nil, err
	}

	var alerts []*model.Alert
	for _, gobj := range geObjList {
		b := []byte(gobj.ResourceData["data"].(string))
		a := &model.Alert{}
		json.Unmarshal(b, a)
		if environment == "" || a.Environment == environment {
			alerts = append(alerts, a)
		}
	}

	return alerts, nil
}

func GetAlert(id string) (*model.Alert, error) {
	data, err := getGenericObjectById("alert", id)
	if err != nil {
		return nil, err
	}

	alert := &model.Alert{}
	err = json.Unmarshal([]byte(data.ResourceData["data"].(string)), alert)
	if err != nil {
		return nil, err
	}

	return alert, nil
}

func CreateAlert(alert *model.Alert) error {
	rclient, err := getRancherClient()
	if err != nil {
		return err
	}

	alert.Id = uuid.Rand().Hex()
	b, err := json.Marshal(*alert)
	if err != nil {
		return err
	}
	resourceData := map[string]interface{}{
		"data": string(b),
	}

	_, err = rclient.GenericObject.Create(&v2client.GenericObject{
		Name:         alert.Id,
		Key:          alert.Id,
		ResourceData: resourceData,
		Kind:         "alert",
	})
	if err != nil {
		return err
	}

	return nil
}

func DeleteAlert(id string) error {
	rclient, err := getRancherClient()
	if err != nil {
		return err
	}

	data, err := getGenericObjectById("alert", id)
	if err != nil {
		logrus.Errorf("Error while getting alert", err)
		return err
	}

	alert := &model.Alert{}
	err = json.Unmarshal([]byte(data.ResourceData["data"].(string)), alert)
	if err != nil {
		return err
	}

	if alert.State != model.AlertStateDisabled {
		return fmt.Errorf("Current state is not inactive, can not perform delete")
	}

	if err = rclient.GenericObject.Delete(&data); err != nil {
		return err
	}

	return nil
}

func UpdateAlert(alert *model.Alert) error {

	rclient, err := getRancherClient()
	if err != nil {
		return err
	}

	alertGO, err := getGenericObjectById("alert", alert.Id)
	if err != nil {
		return err
	}

	b, err := json.Marshal(*alert)
	if err != nil {
		return err
	}
	resourceData := map[string]interface{}{
		"data": string(b),
	}

	_, err = rclient.GenericObject.Update(&alertGO, &v2client.GenericObject{
		Name:         alert.Id,
		Key:          alert.Id,
		ResourceData: resourceData,
		Kind:         "alert",
	})
	if err != nil {
		return err
	}

	return nil
}
