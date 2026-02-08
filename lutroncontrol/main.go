package main

import (
	"flag"
	"os"

	"github.com/unixpickle/essentials"
)

func main() {
	var assetDir string
	var savePath string
	var addr string
	var secret string
	flag.StringVar(&assetDir, "asset-dir", "assets", "path to asset directory")
	flag.StringVar(&savePath, "save-path", "state.json", "path to save server state")
	flag.StringVar(&addr, "addr", ":8080", "address to listen on")
	flag.StringVar(&secret, "secret", "", "secret URL prefix (e.g. somesecret)")
	flag.Parse()

	if _, err := os.Stat(assetDir); os.IsNotExist(err) {
		essentials.Die("The -asset-dir does not exist; pass an -asset-dir argument.")
	}

	username := os.Getenv("LUTRON_USERNAME")
	password := os.Getenv("LUTRON_PASSWORD")
	if username == "" || password == "" {
		essentials.Die("Must specify LUTRON_USERNAME and LUTRON_PASSWORD env vars")
	}

	server, err := NewServer(assetDir, savePath, username, password, secret)
	essentials.Must(err)
	essentials.Must(server.Serve(addr))
}
