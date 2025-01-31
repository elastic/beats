package bbolt

import (
	"encoding/json"
	"os"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	defaultDbPath     = "beat.db" // @TODO define value
	defaultBucketName = "kv"
	defaultDbFileMode = 0o644
)

type BboltValue struct {
	RawValue []byte `json:"rawValue"`
	ExpireAt int64  `json:"expireAt"`
}

type Option func(bbolt *Bbolt)

type Bbolt struct {
	dbPath     string
	dbFileMode os.FileMode
	bucketName string

	db     *bolt.DB
	bucket *bolt.Bucket
}

func New(options ...Option) *Bbolt {
	b := &Bbolt{
		dbPath:     defaultDbPath,
		dbFileMode: defaultDbFileMode,
		bucketName: defaultBucketName,
	}
	for _, opt := range options {
		opt(b)
	}

	return b
}

func WithDbPath(path string) Option {
	return func(b *Bbolt) {
		b.dbPath = path
	}
}

func WithDbFileMode(mode os.FileMode) Option {
	return func(b *Bbolt) {
		b.dbFileMode = mode
	}
}

func WithBucketName(name string) Option {
	return func(b *Bbolt) {
		b.bucketName = name
	}
}

func (b *Bbolt) Connect() error {
	var err error
	b.db, err = initDb(b.dbPath, b.dbFileMode)
	if err != nil {
		return err
	}
	err = b.openBucket()
	if err != nil {
		return err
	}
	return nil
}

func (b *Bbolt) Get(key []byte) ([]byte, error) {
	var returnValue []byte
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(b.bucketName))
		bboltValEncoded := bucket.Get(key)
		if bboltValEncoded == nil { // no value in store
			return nil
		}
		var bboltVal BboltValue
		err := json.Unmarshal(bboltValEncoded, &bboltVal)
		if err != nil {
			return err
		}
		if bboltVal.ExpireAt <= time.Now().UnixNano() { // value expired
			//err = bucket.Delete(key) // since value has expired - no need to keep it in DB
			//if err != nil {
			//	return err
			//}
			return nil
		}
		returnValue = bboltVal.RawValue
		return nil
	})
	if err != nil {
		return nil, err
	}

	return returnValue, nil
}

func (b *Bbolt) Set(key []byte, value []byte, ttl time.Duration) error {
	tx, err := b.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	bucket := tx.Bucket([]byte(b.bucketName))

	bboltValEncoded, err := getMarshalledBboltValue(value, ttl)
	if err != nil {
		return err
	}

	err = bucket.Put(key, bboltValEncoded)
	if err != nil {
		return err
	}

	// Commit the transaction.
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (b *Bbolt) Close() error {
	//TODO more clean up?
	return b.db.Close()
}

func (b *Bbolt) openBucket() error {
	// Ensure the name exists.
	err := b.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(b.bucketName))
		return err
	})
	return err
}

func initDb(path string, mode os.FileMode) (*bolt.DB, error) {
	db, err := bolt.Open(path, mode, nil)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func getMarshalledBboltValue(value []byte, ttl time.Duration) ([]byte, error) {
	return json.Marshal(newBboltValue(value, ttl))
}

func newBboltValue(value []byte, ttl time.Duration) BboltValue {
	return BboltValue{
		RawValue: value,
		ExpireAt: time.Now().UnixNano() + ttl.Nanoseconds(),
	}
}
