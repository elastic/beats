package converter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/elastic/beats/libbeat/common"
)

func TestBytes_Ntohs(t *testing.T) {

	schema := Schema{
		"lastname":    Str("name"),
		"git":         Str("social.github"),
		"number.nine": Int("nine"),
	}

	file, err := os.Open("example.json")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
		return
	}

	var data common.MapStr
	err = json.Unmarshal([]byte(byteValue), &data)
	if err != nil {
		fmt.Println(err)
		return
	}

	conv(schema, data)
}

func conv(schema Schema, data common.MapStr) {

	flatted := data.Flatten()

	event := common.MapStr{}
	for key, mapper := range schema {
		new, _ := mapper.Func(mapper.Key, flatted)
		event.Put(key, new)
	}

	fmt.Println(event.String())
}
