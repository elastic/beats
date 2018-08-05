package replstatus

import (
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type ReplStatusRaw struct {
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

type Member struct {
	Id            bson.ObjectId `bson:"_id,omitempty"`
	Health        bool          `bson:"health"`
	Name          string        `bson:"name"`
	State         int           `bson:"state"`
	StateStr      string        `bson:"stateStr"`
	Uptime        int           `bson:"uptime"`
	OpTime        OpTime        `bson:"optime"`
	OpTimeDate    time.Time     `bson:"optimeDate"`
	ElectionTime  int64         `bson:"electionTime"`
	ElectionDate  time.Time     `bson:"electaionDate"`
	ConfigVersion int           `bson:"configVersion"`
	Self          bool          `bson:"self"`
}

type OpTime struct {
	Ts int64 `bson:"ts"`
	T  int   `bson:"t"`
}

type MemberState int

const (
	STARTUP    MemberState = 0
	PRIMARY    MemberState = 1
	SECONDARY  MemberState = 2
	RECOVERING MemberState = 3
	STARTUP2   MemberState = 5
	UNKNOWN    MemberState = 6
	ARBITER    MemberState = 7
	DOWN       MemberState = 8
	ROLLBACK   MemberState = 9
	REMOVED    MemberState = 10
)

func getReplicationStatus(mongoSession *mgo.Session) (*ReplStatusRaw, error) {
	db := mongoSession.DB("admin")

	var replStatus ReplStatusRaw
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
