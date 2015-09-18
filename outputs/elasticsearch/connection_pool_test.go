package elasticsearch

import (
	"testing"
	"time"

	"github.com/elastic/libbeat/logp"
)

func TestRoundRobin(t *testing.T) {

	var pool ConnectionPool

	urls := []string{"localhost:9200", "localhost:9201"}

	err := pool.SetConnections(urls, "test", "secret")

	if err != nil {
		t.Errorf("Fail to set the connections: %s", err)
	}

	conn := pool.GetConnection()

	if conn.URL != "localhost:9200" {
		t.Errorf("Wrong connection returned: %s", conn.URL)
	}

	conn = pool.GetConnection()
	if conn.URL != "localhost:9201" {
		t.Errorf("Wrong connection returned: %s", conn.URL)
	}
}

func TestMarkDead(t *testing.T) {

	var pool ConnectionPool

	urls := []string{"localhost:9200", "localhost:9201"}

	err := pool.SetConnections(urls, "test", "secret")

	if err != nil {
		t.Errorf("Fail to set the connections: %s", err)
	}

	conn := pool.GetConnection()

	if conn.URL != "localhost:9200" {
		t.Errorf("Wrong connection returned: %s", conn.URL)
	}
	pool.MarkDead(conn)

	conn = pool.GetConnection()
	if conn.URL != "localhost:9201" {
		t.Errorf("Wrong connection returned: %s", conn.URL)
	}

	conn = pool.GetConnection()
	if conn.URL != "localhost:9201" {
		t.Errorf("Wrong connection returned: %s", conn.URL)
	}
	pool.MarkDead(conn)

	conn = pool.GetConnection()
	if conn.URL != "localhost:9201" && conn.URL != "localhost:9200" {
		t.Errorf("No expected connection returned")
	}

}

func TestDeadTimeout(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"elasticsearch"})
	}

	var pool ConnectionPool

	urls := []string{"localhost:9200", "localhost:9201"}

	err := pool.SetConnections(urls, "test", "secret")
	if err != nil {
		t.Errorf("Fail to set the connections: %s", err)
	}
	pool.SetDeadTimeout(10)

	conn := pool.GetConnection()

	if conn.URL != "localhost:9200" {
		t.Errorf("Wrong connection returned: %s", conn.URL)
	}
	pool.MarkDead(conn)
	time.Sleep(10 * time.Second)

	conn = pool.GetConnection()
	if conn.URL != "localhost:9201" {
		t.Errorf("Wrong connection returned: %s", conn.URL)
	}

	conn = pool.GetConnection()
	if conn.URL != "localhost:9200" {
		t.Errorf("Wrong connection returned: %s", conn.URL)
	}
}

func TestMarkLive(t *testing.T) {

	var pool ConnectionPool

	urls := []string{"localhost:9200", "localhost:9201"}

	err := pool.SetConnections(urls, "test", "secret")

	if err != nil {
		t.Errorf("Fail to set the connections: %s", err)
	}

	conn := pool.GetConnection()
	if conn.URL != "localhost:9200" {
		t.Errorf("Wrong connection returned: %s", conn.URL)
	}
	pool.MarkDead(conn)
	pool.MarkLive(conn)

	conn = pool.GetConnection()
	if conn.URL != "localhost:9201" {
		t.Errorf("Wrong connection returned: %s", conn.URL)
	}
	conn = pool.GetConnection()
	if conn.URL != "localhost:9200" {
		t.Errorf("Wrong connection returned: %s", conn.URL)
	}

}
