package util

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
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
