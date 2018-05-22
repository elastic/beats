/*
Package docker is the official Go client for the Docker API.


For more information about the Docker API, see the documentation:
https://docs.docker.com/develop/api

Usage

You use the library by creating a client object and calling methods on it. The
client can be created either from environment variables with NewEnvClient, or
configured manually with NewClient.

For example, to list running containers (the equivalent of "docker ps"):

	package main

	import (
		"context"
		"fmt"

		"docker.io/go-docker"
		"docker.io/go-docker/api/types"
	)

	func main() {
		cli, err := docker.NewEnvClient()
		if err != nil {
			panic(err)
		}

		containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
		if err != nil {
			panic(err)
		}

		for _, container := range containers {
			fmt.Printf("%s %s\n", container.ID[:10], container.Image)
		}
	}
*/
package docker // import "docker.io/go-docker"
