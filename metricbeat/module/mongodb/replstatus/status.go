// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package replstatus

import (
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// MongoReplStatus cointains the status of the replica set from the point of view of the server that processed the command.
type MongoReplStatus struct {
	Ok                      bool      `bson:"ok"`
	Set                     string    `bson:"set"`
	Date                    time.Time `bson:"date"`
	Members                 []Member  `bson:"members"`
	MyState                 int       `bson:"myState"`
	Term                    int       `bson:"term"`
	HeartbeatIntervalMillis int       `bson:"heartbeatIntervalMillis"`
	OpTimes                 struct {
		LastCommitted OpTime `bson:"lastCommittedOpTime"`
		Applied       OpTime `bson:"appliedOpTime"`
		Durable       OpTime `bson:"durableOpTime"`
	} `bson:"optimes"`
}

// Member provides information about a member in the replica set.
type Member struct {
	Health        bool      `bson:"health"`
	Name          string    `bson:"name"`
	State         int       `bson:"state"`
	StateStr      string    `bson:"stateStr"`
	Uptime        int       `bson:"uptime"`
	OpTime        OpTime    `bson:"optime"`
	OpTimeDate    time.Time `bson:"optimeDate"`
	ElectionTime  int64     `bson:"electionTime"`
	ElectionDate  time.Time `bson:"electaionDate"`
	ConfigVersion int       `bson:"configVersion"`
	Self          bool      `bson:"self"`
}

// OpTime holds information regarding the operation from the operation log
type OpTime struct {
	Ts int64 `bson:"ts"` // The timestamp of the last operation applied to this member of the replica set
	T  int   `bson:"t"`  // The term in which the last applied operation was originally generated on the primary.
}

// MemberState shows the state of a member in the replica set
type MemberState int

const (
	// STARTUP state
	STARTUP MemberState = 0
	// PRIMARY state
	PRIMARY MemberState = 1
	// SECONDARY state
	SECONDARY MemberState = 2
	// RECOVERING state
	RECOVERING MemberState = 3
	// STARTUP2 state
	STARTUP2 MemberState = 5
	// UNKNOWN state
	UNKNOWN MemberState = 6
	// ARBITER state
	ARBITER MemberState = 7
	// DOWN state
	DOWN MemberState = 8
	// ROLLBACK state
	ROLLBACK MemberState = 9
	// REMOVED state
	REMOVED MemberState = 10
)

func (optime *OpTime) getTimeStamp() int64 {
	return optime.Ts >> 32
}

func getReplicationStatus(mongoSession *mgo.Session) (*MongoReplStatus, error) {
	db := mongoSession.DB("admin")

	var replStatus MongoReplStatus
	if err := db.Run(bson.M{"replSetGetStatus": 1}, &replStatus); err != nil {
		return nil, err
	}

	return &replStatus, nil
}

func findUnhealthyHosts(members []Member) []string {
	var hosts []string

	for _, member := range members {
		if member.Health == false {
			hosts = append(hosts, member.Name)
		}
	}

	return hosts
}

func findHostsByState(members []Member, state MemberState) []string {
	var hosts []string

	for _, member := range members {
		memberState := MemberState(member.State)
		if memberState == state {
			hosts = append(hosts, member.Name)
		}
	}

	return hosts
}

func findLag(members []Member) (minLag int64, maxLag int64, hasSecondary bool) {
	var minOptime, maxOptime, primaryOptime int64 = 1<<63 - 1, 0, 0
	hasSecondary = false

	for _, member := range members {
		memberState := MemberState(member.State)
		if memberState == SECONDARY {
			hasSecondary = true

			if minOptime > member.OpTime.getTimeStamp() {
				minOptime = member.OpTime.getTimeStamp()
			}

			if member.OpTime.getTimeStamp() > maxOptime {
				maxOptime = member.OpTime.getTimeStamp()
			}
		} else if memberState == PRIMARY {
			primaryOptime = member.OpTime.getTimeStamp()
		}
	}

	minLag = primaryOptime - maxOptime
	maxLag = primaryOptime - minOptime
	return minLag, maxLag, hasSecondary
}

func findOptimesByState(members []Member, state MemberState) []int64 {
	var optimes []int64

	for _, member := range members {
		memberState := MemberState(member.State)
		if memberState == state {
			optimes = append(optimes, member.OpTime.getTimeStamp())
		}
	}

	return optimes
}
