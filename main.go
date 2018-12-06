package main

import (
	"fmt"
	"github.com/keitaro1020/make-graphql-field/cmd"
	"os"
)

func main() {
	if err := cmd.Cmd.Execute(); err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
}
