package replstatus

import (
	"errors"

	"github.com/elastic/beats/libbeat/logp"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type oplogInfo struct {
	allocated int64
	used      float64
	firstTs   int64
	lastTs    int64
	diff      int64
}

// Contains data about collection size
type CollSize struct {
	MaxSize int64   `bson:"maxSize"` // Shows the maximum size of the collection.
	Size    float64 `bson:"size"`    // The total size in memory of all records in a collection.
}

const oplogCol = "oplog.rs"

func getReplicationInfo(mongoSession *mgo.Session) (*oplogInfo, error) {
	// get oplog.rs collection
	db := mongoSession.DB("local")
	if collections, err := db.CollectionNames(); err != nil || !contains(collections, oplogCol) {
		if err == nil {
			err = errors.New("collection oplog.rs was not found")
		}

		logp.Err(err.Error())
		return nil, err
	}
	collection := db.C(oplogCol)

	// get oplog size
	var oplogSize CollSize
	if err := db.Run(bson.D{{Name: "collStats", Value: oplogCol}}, &oplogSize); err != nil {
		return nil, err
	}

	// get first and last items in the oplog
	firstTs, err := getOpTimestamp(collection, "$natural")
	if err != nil {
		return nil, err
	}

	lastTs, err := getOpTimestamp(collection, "-$natural")
	if err != nil {
		return nil, err
	}

	diff := lastTs - firstTs

	return &oplogInfo{
		allocated: oplogSize.MaxSize,
		used:      oplogSize.Size,
		firstTs:   firstTs,
		lastTs:    lastTs,
		diff:      diff,
	}, nil
}

func getOpTimestamp(collection *mgo.Collection, sort string) (int64, error) {
	iter := collection.Find(nil).Sort(sort).Iter()

	var opTime OpTime
	if !iter.Next(&opTime) {
		err := errors.New("objects not found in local.oplog.rs -- Is this a new and empty db instance?")
		logp.Err(err.Error())
		return 0, err
	}

	return opTime.getTimeStamp(), nil
}

func contains(s []string, x string) bool {
	for _, n := range s {
		if x == n {
			return true
		}
	}
	return false
}
