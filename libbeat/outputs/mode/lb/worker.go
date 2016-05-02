package lb

type worker interface {
	run()
}

type WorkerFactory interface {
	count() int // return number of workers
	mk(ctx context) ([]worker, error)
}
