package main

import (
	"fmt"
	"sync"

	"gateman.io/infrastructure/facematch"
)

func main() {
	var wg sync.WaitGroup
	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			res := facematch.VerifyImageQuality(
				"https://t4.ftcdn.net/jpg/02/90/27/39/360_F_290273933_ukYZjDv8nqgpOBcBUo5CQyFcxAzYlZRW.jpg",
			)
			fmt.Println(res)
		}(i)
	}
	wg.Wait()
	// infrastructure.StartServer()
}
