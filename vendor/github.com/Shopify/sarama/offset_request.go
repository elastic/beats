package sarama

type offsetRequestBlock struct {
	time       int64
	maxOffsets int32
}

func (b *offsetRequestBlock) encode(pe packetEncoder) error {
	pe.putInt64(int64(b.time))
	pe.putInt32(b.maxOffsets)
	return nil
}

func (b *offsetRequestBlock) decode(pd packetDecoder) (err error) {
	if b.time, err = pd.getInt64(); err != nil {
		return err
	}
	if b.maxOffsets, err = pd.getInt32(); err != nil {
		return err
	}
	return nil
}

type OffsetRequest struct {
	replicaID *int32
	blocks    map[string]map[int32]*offsetRequestBlock

	storeReplicaID int32
}

func (r *OffsetRequest) encode(pe packetEncoder) error {
	if r.replicaID == nil {
		// default replica ID is always -1 for clients
		pe.putInt32(-1)
	} else {
		pe.putInt32(*r.replicaID)
	}

	err := pe.putArrayLength(len(r.blocks))
	if err != nil {
		return err
	}
	for topic, partitions := range r.blocks {
		err = pe.putString(topic)
		if err != nil {
			return err
		}
		err = pe.putArrayLength(len(partitions))
		if err != nil {
			return err
		}
		for partition, block := range partitions {
			pe.putInt32(partition)
			if err = block.encode(pe); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *OffsetRequest) decode(pd packetDecoder, version int16) error {
	// Ignore replica ID
	if _, err := pd.getInt32(); err != nil {
		return err
	}
	blockCount, err := pd.getArrayLength()
	if err != nil {
		return err
	}
	if blockCount == 0 {
		return nil
	}
	r.blocks = make(map[string]map[int32]*offsetRequestBlock)
	for i := 0; i < blockCount; i++ {
		topic, err := pd.getString()
		if err != nil {
			return err
		}
		partitionCount, err := pd.getArrayLength()
		if err != nil {
			return err
		}
		r.blocks[topic] = make(map[int32]*offsetRequestBlock)
		for j := 0; j < partitionCount; j++ {
			partition, err := pd.getInt32()
			if err != nil {
				return err
			}
			block := &offsetRequestBlock{}
			if err := block.decode(pd); err != nil {
				return err
			}
			r.blocks[topic][partition] = block
		}
	}
	return nil
}

func (r *OffsetRequest) key() int16 {
	return 2
}

func (r *OffsetRequest) version() int16 {
	return 0
}

func (r *OffsetRequest) requiredVersion() KafkaVersion {
	return minVersion
}

func (r *OffsetRequest) SetReplicaID(id int32) {
	r.storeReplicaID = id
	r.replicaID = &r.storeReplicaID
}

func (r *OffsetRequest) AddBlock(topic string, partitionID int32, time int64, maxOffsets int32) {
	if r.blocks == nil {
		r.blocks = make(map[string]map[int32]*offsetRequestBlock)
	}

	if r.blocks[topic] == nil {
		r.blocks[topic] = make(map[int32]*offsetRequestBlock)
	}

	tmp := new(offsetRequestBlock)
	tmp.time = time
	tmp.maxOffsets = maxOffsets

	r.blocks[topic][partitionID] = tmp
}
