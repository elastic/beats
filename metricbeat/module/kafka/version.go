package kafka

import (
	"fmt"

	"github.com/Shopify/sarama"
)

type Version struct {
	String string
}

var (
	minVersion = sarama.V0_8_2_0

	kafkaVersions = map[string]sarama.KafkaVersion{
		"": sarama.V0_8_2_0,

		"0.8.2.0": sarama.V0_8_2_0,
		"0.8.2.1": sarama.V0_8_2_1,
		"0.8.2.2": sarama.V0_8_2_2,
		"0.8.2":   sarama.V0_8_2_2,
		"0.8":     sarama.V0_8_2_2,

		"0.9.0.0": sarama.V0_9_0_0,
		"0.9.0.1": sarama.V0_9_0_1,
		"0.9.0":   sarama.V0_9_0_1,
		"0.9":     sarama.V0_9_0_1,

		"0.10.0.0": sarama.V0_10_0_0,
		"0.10.0.1": sarama.V0_10_0_1,
		"0.10.0":   sarama.V0_10_0_1,
		"0.10.1.0": sarama.V0_10_1_0,
		"0.10.1":   sarama.V0_10_1_0,
		"0.10":     sarama.V0_10_1_0,
	}
)

func (v *Version) Validate() error {
	if _, ok := kafkaVersions[v.String]; !ok {
		return fmt.Errorf("unknown/unsupported kafka vesion '%v'", v.String)
	}
	return nil
}

func (v *Version) Unpack(s string) error {
	tmp := Version{s}
	if err := tmp.Validate(); err != nil {
		return err
	}

	*v = tmp
	return nil
}

func (v *Version) get() sarama.KafkaVersion {
	if v, ok := kafkaVersions[v.String]; ok {
		return v
	}

	return minVersion
}
