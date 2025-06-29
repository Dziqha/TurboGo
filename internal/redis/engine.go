package redis

type Engine struct {
	Memory  *InMemRedis  // from inmem.go
	Storage *PersistentRedis  // from persist.go
}

func NewEngine() (*Engine, error) {
	persist, err := NewPersistent("data/redis.json", false)
	if err != nil {
		return nil, err
	}

	return &Engine{
		Memory:  NewInMem(),
		Storage: persist,
	}, nil
}
