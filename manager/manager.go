package manager

import (
	"encoding/json"
	"net/url"
	"node_manager/conf"
	"node_manager/utils"

	"github.com/pkg/errors"
)

var baseUrl = "http://127.0.0.1:8080"

func UpdateNodeConf(key string, value any) error {
	bytes, err := utils.ReadFromFile(conf.GetConfig().ConfigPath)
	if err != nil {
		return errors.Wrap(err, "read config file error")
	}
	confMap := make(map[string]any)
	err = json.Unmarshal(bytes, &confMap)
	if err != nil {
		errors.Wrap(err, "unmarshal config file error")
	}
	confMap[key] = value
	jbytes, err := json.Marshal(confMap)
	if err != nil {
		return errors.Wrap(err, "marshal config file error")
	}
	return errors.Wrap(utils.WriteToFile(conf.GetConfig().ConfigPath, jbytes), "save config file error")
}

func FlashNodeConf() error {
	data := url.Values{}
	data.Add("action", "reload")
	u, _ := url.JoinPath(baseUrl, conf.GetConfig().Group, "reload")
	_, err := utils.Post(u, data)
	return errors.Wrap(err, "reload config file error")
}

func SyncFile(date string, force bool) error {
	u := baseUrl + "/" + conf.GetConfig().Group + "sync?date=" + date + "&force="
	if force {
		u += "1"
	} else {
		u += "0"
	}
	_, err := utils.Post(u, url.Values{})
	return errors.Wrap(err, "sync file error")
}

func SyncAllFile() error {
	u := baseUrl + "/" + conf.GetConfig().Group + "/repair?force=1"
	_, err := utils.Post(u, url.Values{})
	return errors.Wrap(err, "sync all file error")
}

func FileServiceMonitor() error {
	return nil
}
