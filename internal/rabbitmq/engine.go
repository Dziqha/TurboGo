package rabbitmq


type Engine struct {
	Memory *TaskQueue
	Storage *PersistentTaskQueue
}



func NewEngine() (*Engine, error) {
	persist, err := NewPersistent("data/rabbitmq.json")
	if err != nil {
		return nil, err
	}
	return &Engine{
		Memory:  NewInMem(),
		Storage: persist,
	}, nil
}