package beats

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func Request(url string, client *http.Client) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making http request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	buf, _ := ioutil.ReadAll(resp.Body)
	data := map[string]interface{}{}
	err = json.Unmarshal(buf, &data)

	if err != nil {
		return nil, err
	}

	return data, nil

}
