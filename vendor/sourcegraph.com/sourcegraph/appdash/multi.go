package appdash

// multiStore is like a normal store except all operations occur on the multiple
// underlying stores.
type multiStore struct {
	// stores is the underlying set of stores that operations take place on.
	stores []Store
}

// Collect implements the Collector interface by invoking Collect on each
// underlying store, returning the first error that occurs.
func (ms *multiStore) Collect(id SpanID, anns ...Annotation) error {
	for _, s := range ms.stores {
		if err := s.Collect(id, anns...); err != nil {
			return err
		}
	}
	return nil
}

// Trace implements the Store interface by returning the first trace found by
// asking each underlying store for it in consecutive order.
func (ms *multiStore) Trace(t ID) (*Trace, error) {
	for _, s := range ms.stores {
		trace, err := s.Trace(t)
		if err == ErrTraceNotFound {
			continue
		} else if err != nil {
			return nil, err
		}
		return trace, nil
	}
	return nil, ErrTraceNotFound
}

// MultiStore returns a Store whose operations occur on the multiple given
// stores.
func MultiStore(s ...Store) Store {
	return &multiStore{
		stores: s,
	}
}

// multiStore is like a normal queryer except it queries from multiple
// underlying stores.
type multiQueryer struct {
	// queryers is the underlying set of queryers that operations take place on.
	queryers []Queryer
}

// Traces implements the Queryer interface by returning the union of all
// underlying stores.
//
// It panics if any underlying store does not implement the appdash Queryer
// interface.
func (mq *multiQueryer) Traces(opts TracesOpts) ([]*Trace, error) {
	var (
		union = make(map[ID]struct{})
		all   []*Trace
	)
	for _, q := range mq.queryers {
		traces, err := q.Traces(TracesOpts{})
		if err != nil {
			return nil, err
		}
		for _, t := range traces {
			if _, ok := union[t.ID.Trace]; !ok {
				union[t.ID.Trace] = struct{}{}
				all = append(all, t)
			}
		}
	}
	return all, nil
}

// MultiQueryer returns a Queryer whose Traces method returns a union of all
// traces across each queryer.
func MultiQueryer(q ...Queryer) Queryer {
	return &multiQueryer{
		queryers: q,
	}
}
