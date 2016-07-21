package stats

import (
	"github.com/elastic/beats/metricbeat/module/docker/services/config"
	"github.com/elastic/beats/metricbeat/module/docker/calculator"
	"github.com/elastic/beats/libbeat/logp"
	"sync"
	"errors"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	//"/home/dje/go/src/github.com/elastic/beats/metricbeat/module/docker/vendor/github.com"
	//"github.com/elastic/beats/metricbeat/module/docker/vendor/github.com/fsouza/go-dockerclient"
)

type SoftwareVersion struct {
	major int
	minor int
}
type SocketConfig struct {
	socket    string
	enableTls bool
	caPath    string
	certPath  string
	keyPath   string
}
type DockerStatistics struct {
	socketConfig        SocketConfig
	dockerClient        *docker.Client
	dataGenerator        config.DataGenerator
	dockerConfiguration *config.Config
	dockerVersion        SoftwareVersion
}
var myDockerstat *DockerStatistics
var once sync.Once


func GetInstance ()*DockerStatistics {
	once.Do( func() {
		myDockerstat = &DockerStatistics{}
	})
	return  myDockerstat
}
func CreateDS( config *config.Config) *DockerStatistics {
	ds := GetInstance()
	if ds.dockerConfiguration == nil{
		ds.dockerConfiguration=config
		ds.socketConfig =SocketConfig{
			socket: config.Socket,
			enableTls: config.Tls.Enable,
			caPath: config.Tls.CaPath,
			certPath: config.Tls.CertPath,
			keyPath: config.Tls.KeyPath,
		}
		ds.dockerVersion= SoftwareVersion{
			major:1,
			minor:9,
		}
		logp.Info(" DockerConfig created")
	}else{
		logp.Info(" DockerConfig already exists")
	}
	return ds
}
func InitDockerClient() error{

	var clientErr error
	var err error

	if myDockerstat.dockerClient == nil {
		myDockerstat.dockerClient, clientErr = GetDockerClient()
		myDockerstat.dataGenerator = config.DataGenerator{
			Socket: &myDockerstat.socketConfig.socket,
			CalculatorFactory: calculator.CalculatorFactoryImpl{},
		}
		if clientErr != nil {
			err = errors.New(fmt.Sprint(" Unable to create dockerCLient"))
		}
		logp.Info(" Docker client created")

	} else{
		logp.Info(" Docker client already exist")
	}
	return err;
}
func GetDockerClient() (*docker.Client, error) {

	var client *docker.Client
	var err error
	if myDockerstat.socketConfig.enableTls ==true{
		client, err = docker.NewTLSClient(
			myDockerstat.socketConfig.socket,
			myDockerstat.socketConfig.certPath,
			myDockerstat.socketConfig.keyPath,
			myDockerstat.socketConfig.caPath,
		)
	}else {
		client, err = docker.NewClient(myDockerstat.socketConfig.socket)

	}
	return client, err
}
