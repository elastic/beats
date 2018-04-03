package compose

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"strconv"

	"github.com/docker/libcompose/docker"
	"github.com/docker/libcompose/docker/ctx"
	"github.com/docker/libcompose/project"
	"github.com/docker/libcompose/project/options"
)

// docker-compose project wrapper
type composeProject struct {
	p    project.APIProject
	file string
}

type serviceInfo struct {
	Name    string
	Running bool
	Healthy bool
	// Has been up for too long?:
	Old bool
}

// Regexp matching state to flag container as old
var oldRegexp = regexp.MustCompile("minute")

// EnsureUp starts all the requested services (must be defined in docker-compose.yml)
// with a default timeout of 60 seconds
func EnsureUp(t *testing.T, services ...string) {
	EnsureUpWithTimeout(t, 60, services...)
}

// EnsureUpWithTimeout starts all the requested services (must be defined in docker-compose.yml)
// Wait for `timeout` seconds for health
func EnsureUpWithTimeout(t *testing.T, timeout int, services ...string) {
	// The NO_COMPOSE env variables makes it possible to skip the starting of the environment.
	// This is useful if the service is already running locally.
	if noCompose, err := strconv.ParseBool(os.Getenv("NO_COMPOSE")); err == nil && noCompose {
		return
	}

	compose, err := getComposeProject()
	if err != nil {
		t.Fatal(err)
	}

	// Kill no longer used containers
	compose.KillOld(services)

	for _, service := range services {
		err = compose.Start(service)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Wait for health
	err = compose.Wait(timeout, services...)
	if err != nil {
		t.Fatal(err)
	}
}

// Start the container, unless it's running already
func (c *composeProject) Start(service string) error {
	servicesStatus, err := c.getServices(service)
	if err != nil {
		return err
	}

	if servicesStatus[service] != nil {
		if servicesStatus[service].Running {
			// Someone is running it
			return nil
		}
	}

	c.Lock()
	defer c.Unlock()

	return c.p.Up(context.Background(), options.Up{
		Create: options.Create{
			ForceBuild: true,
		},
	}, service)
}

// Ensure all wanted services are healthy. Wait loop (60s timeout)
func (c *composeProject) Wait(seconds int, services ...string) error {
	healthy := false
	for !healthy && seconds > 0 {
		healthy = true

		servicesStatus, err := c.getServices(services...)
		if err != nil {
			return err
		}

		for _, s := range servicesStatus {
			if !s.Healthy {
				healthy = false
				break
			}
		}

		time.Sleep(1 * time.Second)
		seconds--
	}

	if !healthy {
		return errors.New("Timeout waiting for services to be healthy")
	}
	return nil
}

func (c *composeProject) Kill(service string) {
	c.Lock()
	defer c.Unlock()

	c.p.Kill(context.Background(), "KILL", service)
}

func (c *composeProject) KillOld(except []string) error {
	// Do not kill ourselves or elasticsearch :)
	except = append(except, "beat", "elasticsearch")

	servicesStatus, err := c.getServices()
	if err != nil {
		return err
	}

	for _, s := range servicesStatus {
		// Ignore the ones we want
		if contains(except, s.Name) {
			continue
		}

		if s.Old {
			c.Kill(s.Name)
		}
	}

	return nil
}

// Lock acquires the lock (300s) timeout
// Normally it should only be seconds that the lock is used, but in some cases it can take longer.
func (c *composeProject) Lock() {
	seconds := 300
	for seconds > 0 {
		file, err := os.OpenFile(c.file+".lock", os.O_CREATE|os.O_EXCL, 0500)
		file.Close()
		if err != nil {
			fmt.Println("docker-compose.yml is locked, waiting")
			time.Sleep(1 * time.Second)
			seconds--
			continue
		}
		return
	}

	// This should rarely happen as we lock for start only, less than a second
	panic(errors.New("Timeout waiting for lock, please remove docker-compose.yml.lock"))
}

func (c *composeProject) Unlock() {
	os.Remove(c.file + ".lock")
}

func (c *composeProject) getServices(filter ...string) (map[string]*serviceInfo, error) {
	c.Lock()
	defer c.Unlock()

	result := make(map[string]*serviceInfo)
	services, err := c.p.Ps(context.Background(), filter...)
	if err != nil {
		return nil, err
	}

	containers, err := c.p.Containers(context.Background(), project.Filter{State: project.Running}, filter...)
	if err != nil {
		return nil, err
	}

	for _, c := range services {
		name := strings.Split(c["Name"], "_")[1]
		// In case of several (stopped) containers, always prefer info about running ones
		if result[name] != nil {
			if result[name].Running {
				continue
			}
		}

		service := &serviceInfo{
			Name: name,
		}
		// fill details:
		service.Healthy = strings.Contains(c["State"], "(healthy)")
		service.Running = contains(containers, c["Id"])
		if service.Healthy {
			service.Old = oldRegexp.MatchString(c["State"])
		}
		result[name] = service
	}
	return result, nil
}

func contains(list []string, item string) bool {
	for _, i := range list {
		if item == i {
			return true
		}
	}
	return false
}

func getComposeProject() (*composeProject, error) {
	// find docker-compose
	path, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		if path == "/" {
			return nil, errors.New("docker-compose.yml not found")
		}

		if _, err = os.Stat(path + "/docker-compose.yml"); err != nil {
			path = filepath.Dir(path)
		} else {
			break
		}
	}

	project, err := docker.NewProject(&ctx.Context{
		Context: project.Context{
			ProjectName:  os.Getenv("DOCKER_COMPOSE_PROJECT_NAME"),
			ComposeFiles: []string{path + "/docker-compose.yml"},
		},
	}, nil)
	if err != nil {
		return nil, err
	}

	return &composeProject{project, path + "/docker-compose.yml"}, nil
}
