package main

import (
	"flag"
	"os"

	"github.com/unixpickle/essentials"
)

func main() {
	var savePath string
	var addr string
	flag.StringVar(&savePath, "save-path", "state.json", "path to save server state")
	flag.StringVar(&addr, "addr", ":8080", "address to listen on")
	flag.Parse()

	username := os.Getenv("LUTRON_USERNAME")
	password := os.Getenv("LUTRON_PASSWORD")
	if username == "" || password == "" {
		essentials.Die("Must specify LUTRON_USERNAME and LUTRON_PASSWORD env vars")
	}

	server, err := NewServer(savePath, username, password)
	essentials.Must(err)
	essentials.Must(server.Serve(addr))
}
