package aws

import (
	uuid "github.com/satori/go.uuid"

	"github.com/elastic/beats/libbeat/logp"
)

type txInfo struct {
	FunctionName string
	FunctionArn  string
	ID           uuid.UUID
}

type operation interface {
	Commit(*logp.Logger, *txInfo) error
	Rollback(*logp.Logger, *txInfo) error
}

type transaction struct {
	log        *logp.Logger
	operations []operation
	info       *txInfo
}

func newTransaction(log *logp.Logger, info *txInfo) *transaction {
	return &transaction{
		log:  log.With("function_name", info.FunctionName, "transaction_id", info.ID),
		info: info,
	}
}

func (t *transaction) Add(ops ...operation) {
	t.operations = append(t.operations, ops...)
}

func (t *transaction) Commit() error {
	var err error
	var idx int

	for _, operation := range t.operations {
		err = operation.Commit(t.log, t.info)
		if err != nil {
			break
		}
		idx++
	}

	if err != nil {
		t.log.Debug("and error happened, rolling back previous operations")
		for i := idx - 1; i >= 0; i-- {
			if err := t.operations[i].Rollback(t.log, t.info); err != nil {
				t.log.Errorf("could not rollback operation: %v, error: %s", t.operations[idx], err)
				return err
			}
		}
		return err
	}
	return nil
}
