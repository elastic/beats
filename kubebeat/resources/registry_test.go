package resources

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type numberFetcher struct {
	num        int
	stopCalled bool
}

func newNumberFetcher(num int) Fetcher {
	return &numberFetcher{num, false}
}

func (f *numberFetcher) Fetch(ctx context.Context) ([]FetcherResult, error) {
	return fetchValue(f.num), nil
}

func (f *numberFetcher) Stop() {
	f.stopCalled = true
}

type boolFetcherCondition struct {
	val  bool
	name string
}

func newBoolFetcherCondition(val bool, name string) FetcherCondition {
	return &boolFetcherCondition{val, name}
}

func (c *boolFetcherCondition) Condition() bool {
	return c.val
}

func (c *boolFetcherCondition) Name() string {
	return c.name
}

func fetchValue(num int) []FetcherResult {
	return []FetcherResult{
		{
			Type:     "number",
			Resource: num,
		},
	}
}

func registerNFetchers(t *testing.T, reg FetchersRegistry, n int) {
	for i := 0; i < n; i++ {
		key := fmt.Sprint(i)
		err := reg.Register(key, newNumberFetcher(i))
		assert.NoError(t, err)
	}
}

type RegistryTestSuite struct {
	suite.Suite
	registry FetchersRegistry
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}

func (s *RegistryTestSuite) SetupTest() {
	s.registry = NewFetcherRegistry()
}

func (s *RegistryTestSuite) TestKeys() {
	var tests = []struct {
		key   string
		value int
	}{
		{
			"some_fetcher", 2,
		},
		{
			"other_fetcher", 4,
		},
		{
			"new_fetcher", 6,
		},
	}

	for i, test := range tests {
		f := newNumberFetcher(test.value)
		s.registry.Register(test.key, f)

		s.Equal(i+1, len(s.registry.Keys()))
	}

	keys := s.registry.Keys()

	s.Contains(keys, "some_fetcher")
	s.Contains(keys, "other_fetcher")
	s.Contains(keys, "new_fetcher")
}

func (s *RegistryTestSuite) TestRegisterDuplicateKey() {
	f := newNumberFetcher(1)
	err := s.registry.Register("some-key", f)
	s.NoError(err)

	err = s.registry.Register("some-key", f)
	s.Error(err)
}

func (s *RegistryTestSuite) TestRegister10() {
	count := 10
	registerNFetchers(s.T(), s.registry, count)
	s.Equal(count, len(s.registry.Keys()))
}

func (s *RegistryTestSuite) TestRunNotRegistered() {
	f := newNumberFetcher(1)
	err := s.registry.Register("some-key", f)
	s.NoError(err)

	arr, err := s.registry.Run(context.TODO(), "unknown")
	s.Error(err)
	s.Empty(arr)
}

func (s *RegistryTestSuite) TestRunRegistered() {
	f1 := newNumberFetcher(1)
	err := s.registry.Register("some-key-1", f1)
	s.NoError(err)

	f2 := newNumberFetcher(2)
	err = s.registry.Register("some-key-2", f2)
	s.NoError(err)

	f3 := newNumberFetcher(3)
	err = s.registry.Register("some-key-3", f3)
	s.NoError(err)

	var tests = []struct {
		key   string
		value int
	}{
		{
			"some-key-1", 1,
		},
		{
			"some-key-2", 2,
		},
		{
			"some-key-3", 3,
		},
	}

	for _, test := range tests {
		arr, err := s.registry.Run(context.TODO(), test.key)
		s.NoError(err)
		s.Equal(1, len(arr))
		s.Equal(test.value, arr[0].Resource)
	}
}

func (s *RegistryTestSuite) TestShouldRunNotRegistered() {
	f := newNumberFetcher(1)
	err := s.registry.Register("some-key", f)
	s.NoError(err)

	res := s.registry.ShouldRun("unknown")
	s.False(res)
}

func (s *RegistryTestSuite) TestShouldRun() {
	conditionTrue := newBoolFetcherCondition(true, "always-fetcher-condition")
	conditionFalse := newBoolFetcherCondition(false, "never-fetcher-condition")

	var tests = []struct {
		conditions []FetcherCondition
		expected   bool
	}{
		{
			[]FetcherCondition{}, true,
		},
		{
			[]FetcherCondition{conditionTrue}, true,
		},
		{
			[]FetcherCondition{conditionTrue, conditionTrue}, true,
		},
		{
			[]FetcherCondition{conditionTrue, conditionTrue, conditionFalse}, false,
		},
		{
			[]FetcherCondition{conditionFalse, conditionTrue, conditionTrue, conditionTrue, conditionTrue}, false,
		},
	}

	for _, test := range tests {
		s.registry = NewFetcherRegistry()
		f := newNumberFetcher(1)
		err := s.registry.Register("some-key", f, test.conditions...)
		s.NoError(err)

		should := s.registry.ShouldRun("some-key")
		s.Equal(test.expected, should)
	}
}
