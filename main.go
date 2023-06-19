package main

import (
	"github.com/veritas501/go-elevate-demo/cmd"
	"log"
)

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)
}

func main() {
	cmd.Execute()
}
