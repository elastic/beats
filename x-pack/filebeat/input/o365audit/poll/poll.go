// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package poll

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/x-pack/filebeat/input/o365audit/auth"
)

// Transaction is the interface that wraps a request-response transaction to be
// performed by the poller.
type Transaction interface {
	fmt.Stringer

	// RequestDecorators must return the list of decorators used to customize
	// an http.Request.
	RequestDecorators() []autorest.PrepareDecorator

	// OnResponse receives the resulting http.Response and returns the actions
	// to be performed.
	OnResponse(*http.Response) []Action

	// Delay returns the required delay before performing the request.
	Delay() time.Duration
}

// Poller encapsulates a single-threaded polling loop that performs requests
// and executes actions in response.
type Poller struct {
	decorators []autorest.PrepareDecorator // Fixed decorators to apply to each request.
	log        *logp.Logger
	tp         auth.TokenProvider
	list       transactionList // List of pending transactions.
	interval   time.Duration   // Minimum interval between transactions.
	ctx        context.Context
}

// New creates a new Poller.
func New(options ...PollerOption) (p *Poller, err error) {
	p = &Poller{
		ctx: context.Background(),
	}
	for _, opt := range options {
		if err = opt(p); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// Run starts the poll loop with the given first transaction and continuing with
// any transactions spawned by it. It will execute until an error, a Terminate
// action is returned by a transaction, it runs out of transactions to perform,
// or a context set using WithContext() is done.
func (r *Poller) Run(item Transaction) error {
	r.list.push(item)
	for r.ctx.Err() == nil {
		transaction := r.list.pop()
		if transaction == nil {
			return nil
		}
		if err := r.fetch(transaction); err != nil {
			return err
		}
	}
	return nil
}

func (r *Poller) fetch(item Transaction) error {
	return r.fetchWithDelay(item, r.interval)
}

func (r *Poller) fetchWithDelay(item Transaction, minDelay time.Duration) error {
	r.log.Debugf("* Fetch %s", item)
	// The order here is important. item's decorators must come first as those
	// set the URL, which is required by other decorators (WithQueryParameters).
	decorators := append(
		append([]autorest.PrepareDecorator{}, item.RequestDecorators()...),
		r.decorators...)
	if r.tp != nil {
		token, err := r.tp.Token()
		if err != nil {
			return errors.Wrap(err, "failed getting a token")
		}
		decorators = append(decorators, autorest.WithBearerAuthorization(token))
	}

	request, err := autorest.Prepare(&http.Request{}, decorators...)
	if err != nil {
		return errors.Wrap(err, "failed preparing request")
	}
	delay := max(item.Delay(), minDelay)
	r.log.Debugf(" -- wait %s for %s", delay, request.URL.String())

	response, err := autorest.Send(request,
		autorest.DoCloseIfError(),
		autorest.AfterDelay(delay))
	if err != nil {
		r.log.Warnf("-- error sending request: %v", err)
		return r.fetchWithDelay(item, max(time.Minute, r.interval))
	}

	acts := item.OnResponse(response)
	r.log.Debugf(" <- Result (%s) #acts=%d", response.Status, len(acts))

	for _, act := range acts {
		if err = act(r); err != nil {
			return errors.Wrapf(err, "error acting on %+v", act)
		}
	}

	return nil
}

// Logger returns the logger used.
func (p *Poller) Logger() *logp.Logger {
	return p.log
}

// PollerOption is the type for additional configuration options for a Poller.
type PollerOption func(r *Poller) error

// WithRequestDecorator sets additional request decorators that will be applied
// to all requests.
func WithRequestDecorator(decorators ...autorest.PrepareDecorator) PollerOption {
	return func(r *Poller) error {
		r.decorators = append(r.decorators, decorators...)
		return nil
	}
}

// WithTokenProvider sets the token provider that will be used to set a bearer
// token to all requests.
func WithTokenProvider(tp auth.TokenProvider) PollerOption {
	return func(r *Poller) error {
		if r.tp != nil {
			return errors.New("tried to set more than one token provider")
		}
		r.tp = tp
		return nil
	}
}

// WithLogger sets the logger to use.
func WithLogger(logger *logp.Logger) PollerOption {
	return func(r *Poller) error {
		r.log = logger
		return nil
	}
}

// WithContext sets the context used to terminate the poll loop.
func WithContext(ctx context.Context) PollerOption {
	return func(r *Poller) error {
		r.ctx = ctx
		return nil
	}
}

// WithMinRequestInterval sets the minimum delay between requests.
func WithMinRequestInterval(d time.Duration) PollerOption {
	return func(r *Poller) error {
		r.interval = d
		return nil
	}
}

type listItem struct {
	item Transaction
	next *listItem
}

type transactionList struct {
	head *listItem
	tail *listItem
	size uint
}

func (p *transactionList) push(item Transaction) {
	li := &listItem{
		item: item,
	}
	if p.head != nil {
		p.tail.next = li
	} else {
		p.head = li
	}
	p.tail = li
	p.size++
}

func (p *transactionList) pop() Transaction {
	item := p.head
	if item == nil {
		return nil
	}
	p.head = item.next
	if p.head == nil {
		p.tail = nil
	}
	p.size--
	return item.item
}

// Enqueuer is the interface provided to actions so they can act on a Poller.
type Enqueuer interface {
	Enqueue(item Transaction) error
	RenewToken() error
}

// Action is an operation returned by a transaction.
type Action func(q Enqueuer) error

// Enqueue adds a new transaction to the queue.
func (r *Poller) Enqueue(item Transaction) error {
	r.list.push(item)
	return nil
}

// RenewToken renews the token provider's master token in the case of an
// authorization error.
func (r *Poller) RenewToken() error {
	if r.tp == nil {
		return errors.New("can't renew token: no token provider set")
	}
	return r.tp.Renew()
}

// Terminate action causes the poll loop to finish with the given error.
func Terminate(err error) Action {
	return func(Enqueuer) error {
		if err == nil {
			return errors.New("polling terminated without a specific error")
		}
		return errors.Wrap(err, "polling terminated due to error")
	}
}

// Fetch action will add an element to the transaction queue.
func Fetch(item Transaction) Action {
	return func(q Enqueuer) error {
		return q.Enqueue(item)
	}
}

// RenewToken will renew the token provider's master token in the case of an
// authorization error.
func RenewToken() Action {
	return func(q Enqueuer) error {
		return q.RenewToken()
	}
}

func max(a, b time.Duration) time.Duration {
	if a < b {
		return b
	}
	return a
}
