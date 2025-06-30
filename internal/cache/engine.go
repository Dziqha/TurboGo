package cache

type Engine struct {
	Memory  *InMemCache  // from inmem.go
	Storage *PersistentCache  // from persist.go
}

func NewEngine() (*Engine, error) {
	persist, err := NewPersistent("data/cache.json", false)
	if err != nil {
		return nil, err
	}

	return &Engine{
		Memory:  NewInMem(),
		Storage: persist,
	}, nil
}
