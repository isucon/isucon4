package worker

type Workers []*Worker

func (w Workers) Work() {
	for _, worker := range w {
		worker.Work()
	}
}

func (w Workers) Stop() {
	for _, worker := range w {
		worker.Stop()
	}
}
