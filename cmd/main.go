package main

import (
	"github.com/algolia/fake-insights-generator/pkg/cmd"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	cmd.NewRootCmd().Execute()
}
