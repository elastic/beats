package azure

import (
	"context"
	"fmt"
	"github.com/Azure/azure-amqp-common-go/aad"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"time"

	"github.com/Azure/azure-event-hubs-go"
)

var
(storageAccountName = "mystorageaccount"
	storageAccountKey = "Zm9vCg=="
	// Azure Storage container to store leases and checkpoints
	storageContainerName = "ephcontainer"

	// Azure Event Hub connection string
	eventHubConnStr = "Endpoint=sb://namespace.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=superSecret1234=;EntityPath=hubName")

type azureInput struct {

}


func init() {
	err := input.Register("azure", NewInput)
	if err != nil {
		panic(err)
	}
}

// NewInput creates a new kafka input
func NewInput(
	cfg *common.Config,
	connector channel.Connector,
	inputContext input.Context,
) (input.Input, error) {
	hub, partitions := initHub()
	exit := make(chan struct{})

	handler := func(ctx context.Context, event *eventhub.Event) error {
		text := string(event.Data)
		if text == "exit\n" {
			fmt.Println("Oh snap!! Someone told me to exit!")
			exit <- *new(struct{})
		} else {
			fmt.Println(string(event.Data))
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	for _, partitionID := range partitions {
		_, err := hub.Receive(ctx, partitionID, handler, eventhub.ReceiveWithLatestOffset())
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
	}
	cancel()

	fmt.Println("I am listening...")

	select {
	case <-exit:
		fmt.Println("closing after 2 seconds")
		select {
		case <-time.After(2 * time.Second):
			return
		}
	}
}

// Run runs the input
func (p *azureInput) Run() {
	p.workerOnce.Do(func() {
		visibilityTimeout := int64(p.config.VisibilityTimeout.Seconds())
		regionName, err := getRegionFromQueueURL(p.config.QueueURL)
		if err != nil {
			p.logger.Errorf("failed to get region name from queueURL: %v", p.config.QueueURL)
		}

		awsConfig := p.awsConfig.Copy()
		awsConfig.Region = regionName
		svcSQS := sqs.New(awsConfig)
		svcS3 := s3.New(awsConfig)

		p.workerWg.Add(1)
		go p.run(svcSQS, svcS3, visibilityTimeout)
		p.workerWg.Done()
	})
}

func initHub() (*eventhub.Hub, []string) {
	namespace := mustGetenv("EVENTHUB_NAMESPACE")
	hubMgmt, err := ensureEventHub(context.Background(), HubName)
	if err != nil {

	}

	provider, err := aad.NewJWTProvider(aad.JWTProviderWithEnvironmentVars())
	if err != nil {

	}
	hub, err := eventhub.NewHub(namespace, HubName, provider)
	if err != nil {
		panic(err)
	}
	return hub, *hubMgmt.PartitionIds
}

func mustGetenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("Environment variable '" + key + "' required for integration tests.")
	}
	return v
}

func ensureEventHub(ctx context.Context, name string) (*mgmt.Model, error) {
	namespace := mustGetenv("EVENTHUB_NAMESPACE")
	client := getEventHubMgmtClient()
	hub, err := client.Get(ctx, ResourceGroupName, namespace, name)

	partitionCount := int64(4)
	if err != nil {
		newHub := &mgmt.Model{
			Name: &name,
			Properties: &mgmt.Properties{
				PartitionCount: &partitionCount,
			},
		}

		hub, err = client.CreateOrUpdate(ctx, ResourceGroupName, namespace, name, *newHub)
		if err != nil {
			return nil, err
		}
	}
	return &hub, nil
}

func getEventHubMgmtClient() *mgmt.EventHubsClient {
	subID := mustGetenv("AZURE_SUBSCRIPTION_ID")
	client := mgmt.NewEventHubsClientWithBaseURI(azure.PublicCloud.ResourceManagerEndpoint, subID)
	a, err := azauth.NewAuthorizerFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	client.Authorizer = a
	return &client
}


// Stop stops TCP server
func (p *azureInput) Stop() {

}

// Wait stop the current server
func (p *azureInput) Wait() {
	p.Stop()
}

func createEvent(raw []byte) beat.Event {
	return beat.Event{
		Timestamp: time.Now(),
		Fields: common.MapStr{
			"message": string(raw),
			"log": common.MapStr{
				"source": common.MapStr{

				},
			},
		},
	}
}
