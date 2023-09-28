package main

import (
	"flag"
	"log"
	"os"

	gittheseus "github.com/moznion/git-theseus"
)

func main() {
	var jsonFilePath string
	var dryrun bool

	flag.StringVar(&jsonFilePath, "input-file", "", "[mandatory] a file path to the JSON file")
	flag.BoolVar(&dryrun, "dryrun", false, "a parameter to instruct it to run as dryrun mode (i.e. no destructive operation on git)")

	flag.Parse()

	if jsonFilePath == "" {
		log.Println("[ERROR] the mandatory parameter '-input-file' hasn't been given")
		flag.Usage()
		os.Exit(1)
	}

	err := gittheseus.Run(jsonFilePath, dryrun)
	if err != nil {
		log.Fatal(err)
	}
}
