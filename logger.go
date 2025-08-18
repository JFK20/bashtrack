package main

import (
	"log"
	"os"
)

var (
	ErrorLogger *log.Logger
	InfoLogger  *log.Logger
)

func init() {
	ErrorLogger = log.New(os.Stderr, "Error: ", 0)
	InfoLogger = log.New(os.Stdout, "Info: ", 0)
}
