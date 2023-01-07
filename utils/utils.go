package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

type JsonResult struct {
	Message string      `json:"message"`
	Status  string      `json:"status"`
	Data    interface{} `json:"data"`
}

func WriteToFile(path string, data []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}

func ReadFromFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func Post(u string, data url.Values) (JsonResult, error) {
	var result JsonResult
	resp, err := http.PostForm(u, url.Values{})
	if err != nil {
		return result, err
	}
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return result, err
	}
	if result.Status != "ok" {
		return result, fmt.Errorf("%s", result.Message)
	}
	return result, nil
}
