package main

import "net/http"
import "flag"
import "log"
import "path/filepath"
import "fmt"

func main() {
	var path *string = flag.String("path", ".", "serving files dir path.")
	var port *int = flag.Int("port", 8080, "serving port.")
	flag.Parse()

	var apath, err = filepath.Abs(*path)
	if err != nil {
		log.Println("Error:", err)
	}
	log.Printf("Start Serving HTTP => directory: %s, port: %d", apath, *port)

	var server = http.StripPrefix("/", http.FileServer(http.Dir(*path)))
	http.Handle("/", server)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		log.Println("ListenAndServe:", err)
	}
}
