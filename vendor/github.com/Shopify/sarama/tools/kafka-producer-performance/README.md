# kafka-producer-performance

A command line tool to test producer performance.

### Installation

    go get github.com/Shopify/sarama/tools/kafka-producer-performance


### Usage

    # Display all command line options
    kafka-producer-performance -help
	
	# Minimum invocation
    kafka-producer-performance \
		-brokers=kafka:9092 \
		-message-load=50000 \
		-message-size=100 \
		-topic=producer_test
