package replstatus

import (
	"errors"

	"github.com/elastic/beats/libbeat/logp"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type oplog struct {
	allocated int64
	used      float64
	firstTs   int64
	lastTs    int64
	diff      int64
}

const oplogCol = "oplog.rs"

func getReplicationInfo(mongoSession *mgo.Session) (*oplog, error) {
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

	//  oplog size
	var oplogStatus map[string]interface{}
	if err := db.Run(bson.D{{Name: "collStats", Value: oplogCol}}, &oplogStatus); err != nil {
		return nil, err
	}

	allocated, ok := oplogStatus["maxSize"].(int64)
	if !ok {
		err := errors.New("unexpected maxSize value found in oplog collStats")
		return nil, err
	}

	used, ok := oplogStatus["size"].(float64)
	if !ok {
		err := errors.New("unexpected size value found in oplog collStats")
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

	return &oplog{
		allocated: allocated,
		used:      used,
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
