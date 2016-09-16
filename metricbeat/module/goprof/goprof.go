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
	req, err := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making http request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	buf, _ := ioutil.ReadAll(resp.Body)
	return json.Unmarshal(buf, data)
}
