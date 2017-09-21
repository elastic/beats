package datastore

import (
	"io"
	"os"
	"sync"

	"github.com/boltdb/bolt"

	"github.com/elastic/beats/libbeat/paths"
)

var (
	initDatastoreOnce sync.Once
	ds                *boltDatastore
)

// OpenBucket returns a new Bucket that stores data in {path.data}/beat.db.
// The returned Bucket must be closed when finished to ensure all resources
// are released.
func OpenBucket(name string) (Bucket, error) {
	initDatastoreOnce.Do(func() {
		ds = &boltDatastore{
			path: paths.Resolve(paths.Data, "beat.db"),
			mode: 0600,
		}
	})

	return ds.OpenBucket(name)
}

// Datastore

type Datastore interface {
	OpenBucket(name string) (Bucket, error)
}

type boltDatastore struct {
	mutex    sync.Mutex
	useCount uint32
	path     string
	mode     os.FileMode
	db       *bolt.DB
}

func New(path string, mode os.FileMode) Datastore {
	return &boltDatastore{path: path, mode: mode}
}

func (ds *boltDatastore) OpenBucket(bucket string) (Bucket, error) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	// Initialize the Bolt DB.
	if ds.db == nil {
		var err error
		ds.db, err = bolt.Open(ds.path, ds.mode, nil)
		if err != nil {
			return nil, err
		}
	}

	// Ensure the name exists.
	err := ds.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		return err
	})
	if err != nil {
		return nil, err
	}

	return &boltBucket{ds, bucket}, nil
}

func (ds *boltDatastore) done() {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if ds.useCount > 0 {
		ds.useCount--

		if ds.useCount == 0 {
			ds.db.Close()
			ds.db = nil
		}
	}
}

// Bucket

type Bucket interface {
	io.Closer
	Load(key string, f func(blob []byte) error) error
	Store(key string, blob []byte) error
	Delete(key string) error // Delete removes a key from the bucket. If the key does not exist then nothing is done and a nil error is returned.
	DeleteBucket() error     // Deletes and closes the bucket.
}

// BoltBucket is a Bucket that exposes some Bolt specific APIs.
type BoltBucket interface {
	Bucket
	View(func(tx *bolt.Bucket) error) error
	Update(func(tx *bolt.Bucket) error) error
}

type boltBucket struct {
	ds   *boltDatastore
	name string
}

func (b *boltBucket) Load(key string, f func(blob []byte) error) error {
	return b.ds.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(b.name))

		data := b.Get([]byte(key))
		if data == nil {
			return nil
		}

		return f(data)
	})
}

func (b *boltBucket) Store(key string, blob []byte) error {
	return b.ds.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(b.name))
		return b.Put([]byte(key), blob)
	})
}

func (b *boltBucket) ForEach(f func(key string, blob []byte) error) error {
	return b.ds.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(b.name))

		return b.ForEach(func(k, v []byte) error {
			return f(string(k), v)
		})
	})
}

func (b *boltBucket) Delete(key string) error {
	return b.ds.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(b.name))
		return b.Delete([]byte(key))
	})
}

func (b *boltBucket) DeleteBucket() error {
	err := b.ds.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(b.name))
	})
	b.Close()
	return err
}

func (b *boltBucket) View(f func(*bolt.Bucket) error) error {
	return b.ds.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(b.name))
		return f(b)
	})
}

func (b *boltBucket) Update(f func(*bolt.Bucket) error) error {
	return b.ds.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(b.name))
		return f(b)
	})
}

func (b *boltBucket) Close() error {
	b.ds.done()
	b.ds = nil
	return nil
}
