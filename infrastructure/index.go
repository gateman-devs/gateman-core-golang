package infrastructure

import (
	"sync"

	messagequeue "gateman.io/infrastructure/message_queue"
)

func StartServer() {
	var server serverInterface = &ginServer{}
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		messagequeue.StartQueue()
	}()

	go func() {
		defer wg.Done()
		server.Start()
	}()

	wg.Wait()
}
