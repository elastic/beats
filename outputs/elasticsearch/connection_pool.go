package elasticsearch

import (
	"math"
	"math/rand"
	"time"

	"github.com/elastic/libbeat/logp"
)

type Connection struct {
	Url      string
	Username string
	Password string

	dead       bool
	dead_count int
	timer      *time.Timer
	timeout    time.Duration
}

const (
	default_dead_timeout = 60 //seconds
)

type ConnectionPool struct {
	Connections []*Connection
	rr          int //round robin

	// options
	Dead_timeout time.Duration
}

func (pool *ConnectionPool) SetConnections(urls []string, username string, password string) error {

	var connections []*Connection

	for _, url := range urls {
		conn := Connection{
			Url:      url,
			Username: username,
			Password: password,
		}
		// set default settings
		conn.dead_count = 0
		connections = append(connections, &conn)
	}
	pool.Connections = connections
	pool.rr = -1
	pool.Dead_timeout = default_dead_timeout
	return nil
}

func (pool *ConnectionPool) SetDeadTimeout(timeout int) {
	pool.Dead_timeout = time.Duration(timeout)
}

func (pool *ConnectionPool) SelectRoundRobin() *Connection {

	for count := 0; count < len(pool.Connections); count++ {

		pool.rr += 1
		pool.rr = pool.rr % len(pool.Connections)
		conn := pool.Connections[pool.rr]
		if conn.dead == false {
			return conn
		}
	}

	// no connection is alive, return a random connection
	pool.rr = rand.Intn(len(pool.Connections))
	return pool.Connections[pool.rr]
}

func (pool *ConnectionPool) GetConnection() *Connection {

	if len(pool.Connections) > 1 {
		return pool.SelectRoundRobin()
	}
	// only one connection, no need to select one connection
	return pool.Connections[0]
}

// If a connection fails, it will be marked as dead and put on timeout.
// timeout = default_timeout * 2 ** (fail_count - 1)
// When the timeout is over, the connection will be resurrected and
// returned to the live pool
func (pool *ConnectionPool) MarkDead(conn *Connection) error {

	logp.Debug("elasticsearch", "Mark dead %s", conn.Url)
	conn.dead = true
	conn.dead_count = conn.dead_count + 1
	conn.timeout = pool.Dead_timeout * time.Duration(math.Pow(2, float64(conn.dead_count)-1))
	conn.timer = time.AfterFunc(conn.timeout*time.Second, func() {
		// timeout expires
		conn.dead = false
		logp.Debug("elasticsearch", "Timeout expired. Mark it as alive: %s", conn.Url)
	})

	return nil
}

// A connection that has been previously marked as dead and succeeds will be marked
// as live and the dead_count is set to zero
func (pool *ConnectionPool) MarkLive(conn *Connection) error {
	if conn.dead {
		logp.Debug("elasticsearch", "Mark live %s", conn.Url)
		conn.dead = false
		conn.dead_count = 0
		conn.timer.Stop()
	}
	return nil
}
