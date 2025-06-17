package main

import (
	"fmt"

	"gateman.io/infrastructure/facematch"
)

func main() {
	res:=facematch.Compare("https://res.cloudinary.com/dh3i1wodq/image/upload/v1675417496/cbimage_3_drqdoc.jpg", "https://res.cloudinary.com/dh3i1wodq/image/upload/v1675417496/cbimage_3_drqdoc.jpg")
	fmt.Println(res)
	// infrastructure.StartServer()
}
