package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/sluu99/uuid"
	"github.com/zionwu/monitoring-manager/model"

	v2client "github.com/rancher/go-rancher/v2"
	"github.com/zionwu/monitoring-manager/config"
)

func GetAlertConfig() (*model.AlertConfig, error) {
	geObjList, err := paginateGenericObjects("alertConfig")
	if err != nil {
		logrus.Errorf("fail to list alertConfig,err:%v", err)
		return nil, err
	}

	if len(geObjList) == 0 {
		//init new settings
		return nil, errors.New("Could not find alert config")
	}
	data := geObjList[0]
	config := &model.AlertConfig{}
	if err = json.Unmarshal([]byte(data.ResourceData["data"].(string)), &config); err != nil {
		return nil, err
	}

	return config, nil
}

func CreateOrUpdateAlertConfig(config *model.AlertConfig) (*model.AlertConfig, error) {

	rclient, err := getRancherClient()
	if err != nil {
		return nil, err
	}

	if config.Id == "" {
		config.Id = uuid.Rand().Hex()
	}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	resourceData := map[string]interface{}{
		"data": string(b),
	}

	geObjList, err := paginateGenericObjects("alertConfig")
	if err != nil {
		logrus.Errorf("fail to list alertConfig,err:%v", err)
		return nil, err
	}

	if len(geObjList) == 0 {

		//not exist,create a setting object
		_, err := rclient.GenericObject.Create(&v2client.GenericObject{
			Name:         "alertConfig",
			Key:          "alertConfig",
			ResourceData: resourceData,
			Kind:         "alertConfig",
		})

		if err != nil {
			return nil, fmt.Errorf("Save alert config got error: %v", err)
		}

	} else {
		existing := geObjList[0]

		_, err = rclient.GenericObject.Update(&existing, &v2client.GenericObject{
			Name:         "alertConfig",
			Key:          "alertConfig",
			ResourceData: resourceData,
			Kind:         "alertConfig",
		})
		if err != nil {
			return nil, fmt.Errorf("Save alert config got error: %v", err)
		}
	}

	return config, nil
}

func paginateGenericObjects(kind string) ([]v2client.GenericObject, error) {
	result := []v2client.GenericObject{}
	limit := "1000"
	marker := ""
	var pageData []v2client.GenericObject
	var err error
	for {
		logrus.Debugf("paging got:%v,%v,%v", kind, limit, marker)
		pageData, marker, err = getGenericObjects(kind, limit, marker)
		if err != nil {
			logrus.Debugf("get genericobject err:%v", err)
			return nil, err
		}
		result = append(result, pageData...)
		if marker == "" {
			break
		}
	}
	return result, nil
}

func getGenericObjectById(kind string, id string) (v2client.GenericObject, error) {

	rclient, err := getRancherClient()
	if err != nil {
		return v2client.GenericObject{}, err
	}

	filters := make(map[string]interface{})
	filters["key"] = id
	filters["kind"] = kind
	goCollection, err := rclient.GenericObject.List(&v2client.ListOpts{
		Filters: filters,
	})

	if err != nil {
		logrus.Errorf("Error %v filtering genericObjects by key", err)
		return v2client.GenericObject{}, err
	}

	if len(goCollection.Data) == 0 {
		return v2client.GenericObject{}, fmt.Errorf("can not find the recipient for id %s", id)
	}

	return goCollection.Data[0], nil
}

func getGenericObjects(kind string, limit string, marker string) ([]v2client.GenericObject, string, error) {

	rclient, err := getRancherClient()
	if err != nil {
		return nil, "", err
	}

	filters := make(map[string]interface{})
	filters["kind"] = kind
	filters["limit"] = limit
	filters["marker"] = marker
	goCollection, err := rclient.GenericObject.List(&v2client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		logrus.Errorf("fail querying generic objects, error:%v", err)
		return nil, "", err
	}
	//get next marker
	nextMarker := ""
	if goCollection.Pagination != nil && goCollection.Pagination.Next != "" {
		r, err := url.Parse(goCollection.Pagination.Next)
		if err != nil {
			logrus.Errorf("fail parsing next url, error:%v", err)
			return nil, "", err
		}
		nextMarker = r.Query().Get("marker")
	}
	return goCollection.Data, nextMarker, err

}

func getRancherClient() (*v2client.RancherClient, error) {
	c := config.GetConfig()
	url := fmt.Sprintf("%s/v2-beta/schemas", c.CattleURL)
	rclient, err := v2client.NewRancherClient(&v2client.ClientOpts{
		Timeout:   time.Second * 30,
		Url:       url,
		AccessKey: c.CattleAccessKey,
		SecretKey: c.CattleSecretKey,
	})
	if err != nil {
		return nil, err
	}

	return rclient, nil
}
