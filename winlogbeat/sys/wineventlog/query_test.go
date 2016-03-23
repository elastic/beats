// +build !integration

package wineventlog

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func ExampleQuery() {
	q, _ := Query{Log: "System", EventID: "10, 200-500, -311", Level: "info"}.Build()
	fmt.Println(q)
	// Output: <QueryList>
	//   <Query Id="0">
	//     <Select Path="System">*[System[(EventID=10 or (EventID &gt;= 200 and EventID &lt;= 500)) and (Level = 0 or Level = 4)]]</Select>
	//     <Suppress Path="System">*[System[(EventID=311)]]</Suppress>
	//   </Query>
	// </QueryList>
}

func TestIgnoreOlderQuery(t *testing.T) {
	const expected = `<QueryList>
  <Query Id="0">
    <Select Path="Application">*[System[TimeCreated[timediff(@SystemTime) &lt;= 3600000]]]</Select>
  </Query>
</QueryList>`

	q, err := Query{Log: "Application", IgnoreOlder: time.Hour}.Build()
	if assert.NoError(t, err) {
		assert.Equal(t, expected, q)
		fmt.Println(q)
	}
}

func TestEventIDQuery(t *testing.T) {
	const expected = `<QueryList>
  <Query Id="0">
    <Select Path="Application">*[System[(EventID=1 or (EventID &gt;= 1 and EventID &lt;= 100))]]</Select>
    <Suppress Path="Application">*[System[(EventID=75)]]</Suppress>
  </Query>
</QueryList>`

	q, err := Query{Log: "Application", EventID: "1, 1-100, -75"}.Build()
	if assert.NoError(t, err) {
		assert.Equal(t, expected, q)
		fmt.Println(q)
	}
}

func TestLevelQuery(t *testing.T) {
	const expected = `<QueryList>
  <Query Id="0">
    <Select Path="Application">*[System[(Level = 5)]]</Select>
  </Query>
</QueryList>`

	q, err := Query{Log: "Application", Level: "Verbose"}.Build()
	if assert.NoError(t, err) {
		assert.Equal(t, expected, q)
		fmt.Println(q)
	}
}

func TestProviderQuery(t *testing.T) {
	const expected = `<QueryList>
  <Query Id="0">
    <Select Path="Application">*[System[Provider[@Name='mysrc']]]</Select>
  </Query>
</QueryList>`

	q, err := Query{Log: "Application", Provider: []string{"mysrc"}}.Build()
	if assert.NoError(t, err) {
		assert.Equal(t, expected, q)
		fmt.Println(q)
	}
}

func TestCombinedQuery(t *testing.T) {
	const expected = `<QueryList>
  <Query Id="0">
    <Select Path="Application">*[System[TimeCreated[timediff(@SystemTime) &lt;= 3600000] and (EventID=1 or (EventID &gt;= 1 and EventID &lt;= 100)) and (Level = 3)]]</Select>
    <Suppress Path="Application">*[System[(EventID=75)]]</Suppress>
  </Query>
</QueryList>`

	q, err := Query{
		Log:         "Application",
		IgnoreOlder: time.Hour,
		EventID:     "1, 1-100, -75",
		Level:       "Warning",
	}.Build()
	if assert.NoError(t, err) {
		assert.Equal(t, expected, q)
		fmt.Println(q)
	}
}

func TestQueryNoParams(t *testing.T) {
	const expected = `<QueryList>
  <Query Id="0">
    <Select Path="Application">*</Select>
  </Query>
</QueryList>`

	q, err := Query{Log: "Application"}.Build()
	if assert.NoError(t, err) {
		assert.Equal(t, expected, q)
		fmt.Println(q)
	}
}
