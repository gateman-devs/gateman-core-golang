package main

import (
	"gateman.io/infrastructure"
	"gateman.io/infrastructure/env"
)

func init() {
	env.LoadEnv()
}

func main() {
	infrastructure.StartServer()
}
