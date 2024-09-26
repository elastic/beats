package customProvider

import (
	"context"
	"fmt"
	"testing"
)

func TestBeatProvider(t *testing.T) {
	p := provider{}
	fmt.Println(p.Retrieve(context.Background(), "filebeat:/Users/khushijain/Documents/beats/x-pack/filebeat/filebeat.yml", nil))

}
