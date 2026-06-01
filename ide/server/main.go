package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"
)

func main() {
	projectRoot := flag.String("project", filepath.Join("..", "testdata", "sample-project"), "project root to edit and execute")
	staticRoot := flag.String("static", filepath.Join("..", "static"), "directory containing IDE static assets")
	artifactRoot := flag.String("artifacts", "", "directory used to persist IDE artifacts")
	listenAddr := flag.String("listen", "127.0.0.1:8080", "address to listen on")
	flag.Parse()

	server, err := NewServer(*projectRoot, *staticRoot, *artifactRoot)
	if err != nil {
		log.Fatalf("create IDE server: %v", err)
	}

	log.Printf("picoceci IDE listening on http://%s", *listenAddr)
	log.Printf("project root: %s", server.projectRoot)
	if err := http.ListenAndServe(*listenAddr, server.Handler()); err != nil {
		log.Fatalf("serve IDE: %v", err)
	}
}
