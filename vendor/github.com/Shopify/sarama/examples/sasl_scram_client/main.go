package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/Shopify/sarama"
)

func init() {
	sarama.Logger = log.New(os.Stdout, "[Sarama] ", log.LstdFlags)
}

var (
	brokers   = flag.String("brokers", os.Getenv("KAFKA_PEERS"), "The Kafka brokers to connect to, as a comma separated list")
	userName  = flag.String("username", "", "The SASL username")
	passwd    = flag.String("passwd", "", "The SASL password")
	algorithm = flag.String("algorithm", "", "The SASL SCRAM SHA algorithm sha256 or sha512 as mechanism")
	topic     = flag.String("topic", "default_topic", "The Kafka topic to use")
	certFile  = flag.String("certificate", "", "The optional certificate file for client authentication")
	keyFile   = flag.String("key", "", "The optional key file for client authentication")
	caFile    = flag.String("ca", "", "The optional certificate authority file for TLS client authentication")
	verifySSL = flag.Bool("verify", false, "Optional verify ssl certificates chain")
	useTLS    = flag.Bool("tls", false, "Use TLS to communicate with the cluster")
	mode      = flag.String("mode", "produce", "Mode to run in: \"produce\" to produce, \"consume\" to consume")
	logMsg    = flag.Bool("logmsg", false, "True to log consumed messages to console")

	logger = log.New(os.Stdout, "[Producer] ", log.LstdFlags)
)

func createTLSConfiguration() (t *tls.Config) {
	t = &tls.Config{
		InsecureSkipVerify: *verifySSL,
	}
	if *certFile != "" && *keyFile != "" && *caFile != "" {
		cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			log.Fatal(err)
		}

		caCert, err := ioutil.ReadFile(*caFile)
		if err != nil {
			log.Fatal(err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		t = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
			InsecureSkipVerify: *verifySSL,
		}
	}
	return t
}

func main() {
	flag.Parse()

	if *brokers == "" {
		log.Fatalln("at least one broker is required")
	}
	splitBrokers := strings.Split(*brokers, ",")

	if *userName == "" {
		log.Fatalln("SASL username is required")
	}

	if *passwd == "" {
		log.Fatalln("SASL password is required")
	}

	conf := sarama.NewConfig()
	conf.Producer.Retry.Max = 1
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Return.Successes = true
	conf.Metadata.Full = true
	conf.Version = sarama.V0_10_0_0
	conf.ClientID = "sasl_scram_client"
	conf.Metadata.Full = true
	conf.Net.SASL.Enable = true
	conf.Net.SASL.User = *userName
	conf.Net.SASL.Password = *passwd
	conf.Net.SASL.Handshake = true
	if *algorithm == "sha512" {
		conf.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA512} }
		conf.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA512)
	} else if *algorithm == "sha256" {
		conf.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA256} }
		conf.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA256)

	} else {
		log.Fatalf("invalid SHA algorithm \"%s\": can be either \"sha256\" or \"sha512\"", *algorithm)
	}

	if *useTLS {
		conf.Net.TLS.Enable = true
		conf.Net.TLS.Config = createTLSConfiguration()
	}

	if *mode == "consume" {
		consumer, err := sarama.NewConsumer(splitBrokers, conf)
		if err != nil {
			panic(err)
		}
		log.Println("consumer created")
		defer func() {
			if err := consumer.Close(); err != nil {
				log.Fatalln(err)
			}
		}()
		log.Println("commence consuming")
		partitionConsumer, err := consumer.ConsumePartition(*topic, 0, sarama.OffsetOldest)
		if err != nil {
			panic(err)
		}

		defer func() {
			if err := partitionConsumer.Close(); err != nil {
				log.Fatalln(err)
			}
		}()

		// Trap SIGINT to trigger a shutdown.
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)

		consumed := 0
	ConsumerLoop:
		for {
			log.Println("in the for")
			select {
			case msg := <-partitionConsumer.Messages():
				log.Printf("Consumed message offset %d\n", msg.Offset)
				if *logMsg {
					log.Printf("KEY: %s VALUE: %s", msg.Key, msg.Value)
				}
				consumed++
			case <-signals:
				break ConsumerLoop
			}
		}

		log.Printf("Consumed: %d\n", consumed)

	} else {
		syncProducer, err := sarama.NewSyncProducer(splitBrokers, conf)
		if err != nil {
			logger.Fatalln("failed to create producer: ", err)
		}
		partition, offset, err := syncProducer.SendMessage(&sarama.ProducerMessage{
			Topic: *topic,
			Value: sarama.StringEncoder("test_message"),
		})
		if err != nil {
			logger.Fatalln("failed to send message to ", *topic, err)
		}
		logger.Printf("wrote message at partition: %d, offset: %d", partition, offset)
		_ = syncProducer.Close()
	}
	logger.Println("Bye now !")

}
