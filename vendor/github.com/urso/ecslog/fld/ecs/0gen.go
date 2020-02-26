package ecs

//go:generate -command genfields go run $GOPATH/src/github.com/urso/ecslog/cmd/genfields/main.go

//go:generate genfields -out schema.go -fmt -version 1.0.0 -schema $GOPATH/src/github.com/elastic/ecs/schemas
