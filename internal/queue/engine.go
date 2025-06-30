package queue


type Engine struct {
	Memory *TaskQueue
	Storage *PersistentTaskQueue
}



func NewEngine() (*Engine, error) {
	persist, err := NewPersistent("data/queue.json")
	if err != nil {
		return nil, err
	}
	return &Engine{
		Memory:  NewInMem(),
		Storage: persist,
	}, nil
}