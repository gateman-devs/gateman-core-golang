package main

import (
	"fmt"

	"gateman.io/infrastructure/facematch"
)

func main() {
	res:=facematch.Compare("https://media.cnn.com/api/v1/images/stellar/prod/230711141746-01-mark-zuckerberg-life-in-pictures-lead-restricted.jpg?q=w_3000,c_fill", "https://media.cnn.com/api/v1/images/stellar/prod/230711141746-01-mark-zuckerberg-life-in-pictures-lead-restricted.jpg?q=w_3000,c_fill")
	fmt.Println(res)
	// infrastructure.StartServer()
}
