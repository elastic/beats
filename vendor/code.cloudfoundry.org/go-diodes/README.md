![diode][diode-logo]

[![GoDoc][go-doc-badge]][go-doc] [![travis][travis-badge]][travis]

Diodes are ring buffers manipulated via atomics.

Diodes are optimized for high throughput scenarios where losing data is
acceptable. Unlike a channel, a diode will overwrite data on writes in lieu
of blocking. A diode does its best to not "push back" on the producer.
In other words, invoking `Set()` on a diode never blocks.

### Installation

```bash
go get code.cloudfoundry.org/go-diodes
```

### Example: Basic Use

```go
d := diodes.NewOneToOne(1024, diodes.AlertFunc(func(missed int) {
	log.Printf("Dropped %d messages", missed)
}))

// writer
go func() {
	for i := 0; i < 2048; i++ {
		// Warning: Do not use i. By taking the address,
		// you would not get each value
		j := i
		d.Set(diodes.GenericDataType(&j))
	}
}()

// reader
poller := diodes.NewPoller(d)
for {
	i := poller.Next()
	fmt.Println(*(*int)(i))
}
```

### Example: Creating a Concrete Shell

Diodes accept and return `diodes.GenericDataType`. It is recommended to not
use these generic pointers directly. Rather, it is a much better experience to
wrap the diode in a concrete shell that accepts the types your program works
with and does the type casting for you. Here is an example of how to create a
concrete shell for `[]byte`:

```go
type OneToOne struct {
	d *diodes.Poller
}

func NewOneToOne(size int, alerter diodes.Alerter) *OneToOne {
	return &OneToOne{
		d: diodes.NewPoller(diodes.NewOneToOne(size, alerter)),
	}
}

func (d *OneToOne) Set(data []byte) {
	d.d.Set(diodes.GenericDataType(&data))
}

func (d *OneToOne) TryNext() ([]byte, bool) {
	data, ok := d.d.TryNext()
	if !ok {
		return nil, ok
	}

	return *(*[]byte)(data), true
}

func (d *OneToOne) Next() []byte {
	data := d.d.Next()
	return *(*[]byte)(data)
}
```

Creating a concrete shell gives you the following advantages:

- The compiler will tell you if you use a diode to read or write data of the
  wrong type.
- The type casting syntax in go is not common and should be hidden.
- It prevents the generic pointer type from escaping in to client code.

### Dropping Data

The diode takes an `Alerter` as an argument to alert the user code to when
the read noticed it missed data. It is important to note that the go-routine
consuming from the diode is used to signal the alert.

When the diode notices it has fallen behind, it will move the read index to
the new write index and therefore drop more than a single message.

There are two things to consider when choosing a diode:

1. Storage layer
2. Access layer

### Storage Layer

##### OneToOne

The OneToOne diode is meant to be used by one producing (invoking `Set()`)
go-routine and a (different) consuming (invoking `TryNext()`) go-routine. It
is not thread safe for multiple readers or writers.

##### ManyToOne

The ManyToOne diode is optimized for many producing (invoking `Set()`)
go-routines and a single consuming (invoking `TryNext()`) go-routine. It is
not thread safe for multiple readers.

It is recommended to have a larger diode buffer size if the number of producers
is high. This is to avoid the diode from having to mitigate write collisions
(it will call its alert function if this occurs).

### Access Layer

##### Poller

The Poller uses polling via `time.Sleep(...)` when `Next()` is invoked. While
polling might seem sub-optimal, it allows the producer to be completely
decoupled from the consumer. If you require very minimal push back on the
producer, then the Poller is a better choice. However, if you require several
diodes (e.g. one per connected client), then having several go-routines
polling (sleeping) may be hard on the scheduler.

##### Waiter

The Waiter uses a conditional mutex to manage when the reader is alerted
of new data. While this method is great for the scheduler, it does have
extra overhead for the producer. Therefore, it is better suited for situations
where you have several diodes and can afford slightly slower producers.

### Benchmarks

There are benchmarks that compare the various storage and access layers to
channels. To run them:

```
go test -bench=. -run=NoTest
```

### Known Issues

If a diode was to be written to `18446744073709551615+1` times it would overflow
a `uint64`. This will cause problems if the size of the diode is not a power
of two (`2^x`). If you write into a diode at the rate of one message every
nanosecond, without restarting your process, it would take you 584.54 years to
encounter this issue.

[diode-logo]:   https://raw.githubusercontent.com/cloudfoundry/go-diodes/gh-pages/diode-logo.png
[go-doc-badge]: https://godoc.org/code.cloudfoundry.org/go-diodes?status.svg
[go-doc]:       https://godoc.org/code.cloudfoundry.org/go-diodes
[travis-badge]: https://travis-ci.org/cloudfoundry/go-diodes.svg?branch=master
[travis]:       https://travis-ci.org/cloudfoundry/go-diodes?branch=master
