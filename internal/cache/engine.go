package cache

type Engine struct {
	Memory *InMemCache // from inmem.go
}

func NewEngine() (*Engine, error) {
	return &Engine{
		Memory: NewInMem(),
	}, nil
}
