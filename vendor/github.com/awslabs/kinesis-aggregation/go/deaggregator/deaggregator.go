// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package deaggregator

import (
	"crypto/md5"
	"fmt"

	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/golang/protobuf/proto"

	rec "github.com/awslabs/kinesis-aggregation/go/records"
)

// Magic File Header for a KPL Aggregated Record
var KplMagicHeader = fmt.Sprintf("%q", []byte("\xf3\x89\x9a\xc2"))

const (
	KplMagicLen = 4  // Length of magic header for KPL Aggregate Record checking.
	DigestSize  = 16 // MD5 Message size for protobuf.
)

// DeaggregateRecords takes an array of Kinesis records and expands any Protobuf
// records within that array, returning an array of all records
func DeaggregateRecords(records []*kinesis.Record) ([]*kinesis.Record, error) {
	var isAggregated bool
	allRecords := make([]*kinesis.Record, 0)
	for _, record := range records {
		isAggregated = true

		var dataMagic string
		var decodedDataNoMagic []byte
		// Check if record is long enough to have magic file header
		if len(record.Data) >= KplMagicLen {
			dataMagic = fmt.Sprintf("%q", record.Data[:KplMagicLen])
			decodedDataNoMagic = record.Data[KplMagicLen:]
		} else {
			isAggregated = false
		}

		// Check if record has KPL Aggregate Record Magic Header and data length
		// is correct size
		if KplMagicHeader != dataMagic || len(decodedDataNoMagic) <= DigestSize {
			isAggregated = false
		}

		if isAggregated {
			messageDigest := fmt.Sprintf("%x", decodedDataNoMagic[len(decodedDataNoMagic)-DigestSize:])
			messageData := decodedDataNoMagic[:len(decodedDataNoMagic)-DigestSize]

			calculatedDigest := fmt.Sprintf("%x", md5.Sum(messageData))

			// Check protobuf MD5 hash matches MD5 sum of record
			if messageDigest != calculatedDigest {
				isAggregated = false
			} else {
				aggRecord := &rec.AggregatedRecord{}
				err := proto.Unmarshal(messageData, aggRecord)

				if err != nil {
					return nil, err
				}

				partitionKeys := aggRecord.PartitionKeyTable

				for _, aggrec := range aggRecord.Records {
					newRecord := createUserRecord(partitionKeys, aggrec, record)
					allRecords = append(allRecords, newRecord)
				}
			}
		}

		if !isAggregated {
			allRecords = append(allRecords, record)
		}
	}

	return allRecords, nil
}

// createUserRecord takes in the partitionKeys of the aggregated record, the individual
// deaggregated record, and the original aggregated record builds a kinesis.Record and
// returns it
func createUserRecord(partitionKeys []string, aggRec *rec.Record, record *kinesis.Record) (*kinesis.Record) {
	partitionKey := partitionKeys[*aggRec.PartitionKeyIndex]

	return &kinesis.Record{
		ApproximateArrivalTimestamp: record.ApproximateArrivalTimestamp,
		Data: aggRec.Data,
		EncryptionType: record.EncryptionType,
		PartitionKey: &partitionKey,
		SequenceNumber: record.SequenceNumber,
	}
}
