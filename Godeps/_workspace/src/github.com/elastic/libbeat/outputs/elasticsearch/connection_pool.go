package elasticsearch

import (
	"math"
	"math/rand"
	"time"

	"github.com/elastic/libbeat/logp"
)

type Connection struct {
	URL      string
	Username string
	Password string

	dead      bool
	deadCount int
	timer     *time.Timer
}

const (
	defaultDeadTimeout time.Duration = 60 * time.Second
)

type ConnectionPool struct {
	Connections []*Connection
	rr          int //round robin

	// options
	DeadTimeout time.Duration
}

func (pool *ConnectionPool) SetConnections(urls []string, username string, password string) error {

	var connections []*Connection

	for _, url := range urls {
		conn := Connection{
			URL:      url,
			Username: username,
			Password: password,
		}
		// set default settings
		conn.deadCount = 0
		connections = append(connections, &conn)
	}
	pool.Connections = connections
	pool.rr = -1
	pool.DeadTimeout = defaultDeadTimeout
	return nil
}

func (pool *ConnectionPool) SetDeadTimeout(timeout time.Duration) {
	pool.DeadTimeout = timeout
}

func (pool *ConnectionPool) selectRoundRobin() *Connection {

	for count := 0; count < len(pool.Connections); count++ {

		pool.rr++
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

// GetConnection finds a live connection.
func (pool *ConnectionPool) GetConnection() *Connection {

	if len(pool.Connections) > 1 {
		return pool.selectRoundRobin()
	}

	// only one connection, no need to select one connection
	// TODO(urso): we want to return nil if connection is not live?
	return pool.Connections[0]
}

// MarkDead marks a failed connection as dead and put on timeout.
// timeout = DeadTimeout * 2 ^ (deadCount - 1)
// When the timeout is over, the connection will be resurrected and
// returned to the live pool.
func (pool *ConnectionPool) MarkDead(conn *Connection) {

	if !conn.dead {
		logp.Debug("elasticsearch", "Mark dead %s", conn.URL)
		conn.dead = true
		conn.deadCount = conn.deadCount + 1
		timeout := pool.DeadTimeout * time.Duration(math.Pow(2, float64(conn.deadCount)-1))
		conn.timer = time.AfterFunc(timeout*time.Second, func() {
			// timeout expires
			conn.dead = false
			logp.Debug("elasticsearch", "Timeout expired. Mark it as alive: %s", conn.URL)
		})
	}
}

// MarkLive marks a connection as live if the connection has been previously
// marked as dead and succeeds.
func (pool *ConnectionPool) MarkLive(conn *Connection) {
	if conn.dead {
		logp.Debug("elasticsearch", "Mark live %s", conn.URL)
		conn.dead = false
		conn.deadCount = 0
		conn.timer.Stop()
	}
}
