package salesforce

import v2 "github.com/elastic/beats/v7/filebeat/input/v2"

// compile-time check if querier implements InputManager
var _ v2.InputManager = InputManager{}
