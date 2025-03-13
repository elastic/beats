package v2

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// TestNewContextParameters ensures when new fields are added to the v2.Contest
// they also are added to the contractor or the decision of not doing so is
// explicit.
func TestNewContextParameters(t *testing.T) {
	ctx := NewContext(
		"test-id",
		"test-id-without-name",
		"test-name",
		beat.Info{Beat: "test-beat"},
		context.Background(),
		noopStatusReporter{},
		monitoring.NewRegistry(),
		logp.NewLogger("test"),
	)

	v := reflect.ValueOf(ctx)
	typeOfCtx := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := typeOfCtx.Field(i).Name

		// ignore unexported fields
		if !field.CanSet() {
			continue
		}

		assert.Falsef(t, field.IsZero(),
			"v2.Context field %s was not set by the constructor. A new field"+
				"might have been added, please consider if you need to change "+
				"the constructor or to skip the field in this test",
			fieldName)
	}
}

type noopStatusReporter struct{}

func (n noopStatusReporter) UpdateStatus(status.Status, string) {
}
