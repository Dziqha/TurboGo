package main

type World struct {
	ID           int32 `json:"id"`
	RandomNumber int32 `json:"randomNumber"`
}

type Fortune struct {
	ID      int32  `json:"id"`
	Message string `json:"message"`
}

type Message struct {
	Message string `json:"message"`
}
