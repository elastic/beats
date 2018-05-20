package mtest

import (
	"net"
	"os"
)

// GetIntegrationConfig generates a base configuration with common values for
// integration tests
func GetIntegrationConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":   "rabbitmq",
		"hosts":    getTestRabbitMQHost(),
		"username": getTestRabbitMQUsername(),
		"password": getTestRabbitMQPassword(),
	}
}

const (
	rabbitmqDefaultHost     = "localhost"
	rabbitmqDefaultPort     = "15672"
	rabbitmqDefaultUsername = "guest"
	rabbitmqDefaultPassword = "guest"
)

func getTestRabbitMQHost() string {
	return net.JoinHostPort(
		getenv("RABBITMQ_HOST", rabbitmqDefaultHost),
		getenv("RABBITMQ_PORT", rabbitmqDefaultPort),
	)
}

func getTestRabbitMQUsername() string {
	return getenv("RABBITMQ_USERNAME", rabbitmqDefaultUsername)
}

func getTestRabbitMQPassword() string {
	return getenv("RABBITMQ_PASSWORD", rabbitmqDefaultPassword)
}

func getenv(name, defaultValue string) string {
	return strDefault(os.Getenv(name), defaultValue)
}

func strDefault(a, defaults string) string {
	if len(a) == 0 {
		return defaults
	}
	return a
}
