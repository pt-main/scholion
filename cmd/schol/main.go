package main

import (
	"fmt"
	"github.com/pt-main/scholion/cmd/cli"
)

func main() {
	if err := cli.New().Main(); err != nil {
		fmt.Println(err)
	}
}
