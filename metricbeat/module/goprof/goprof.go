package goprof

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func Request(url string, client *http.Client) (map[string]interface{}, error) {
	data := map[string]interface{}{}
	err := RequestInto(&data, url, client)
	return data, err
}

func RequestInto(data interface{}, url string, client *http.Client) error {
	buf, err := RequestLoad(url, client)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, data)
}

func RequestLoad(url string, client *http.Client) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making http request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	return ioutil.ReadAll(resp.Body)
}
