package prospector

import "github.com/elastic/beats/filebeat/input"

type Prospectorer = input.Prospectorer
type Prospector = input.Prospector

type Context = input.Context

type Factory = input.Factory

var Register = input.Register
var GetFactory = input.GetFactory
var New = input.New
var NewRunnerFactory = input.NewRunnerFactory
