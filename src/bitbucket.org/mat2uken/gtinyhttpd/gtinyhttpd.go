package main

import "net/http"
import "flag"
import "log"
import "path/filepath"
import "fmt"
import "os"
import "os/exec"
import "io/ioutil"
import "bufio"
import "strings"
import "runtime"

const (
	//	HostFilePath = "/Users/ku/Desktop/gtinyhttpd/hosts"
	HostFilePath = "/private/etc/hosts"
)

func ReadHostsFile() {

}

func main() {
	var add_hosts *string = flag.String("add-hosts", "", "add entry to hosts.")
	var del_hosts *string = flag.String("del-hosts", "", "del entry to hosts.")
	var path *string = flag.String("path", ".", "serving files dir path.")
	var port *int = flag.Int("port", 8080, "serving port.")
	flag.Parse()

	if *add_hosts != "" || *del_hosts != "" {
		var uid = os.Getuid()
		if uid != 0 {
			log.Printf("are you root? => uid: %d", uid)
			os.Exit(1)
		}

		var lines []string
		f, err := os.Open(HostFilePath)
		if err != nil {
			panic(err)
		}
		r := bufio.NewScanner(f)
		for r.Scan() {
			line := r.Text()
			if strings.Index(line, "#") == 0 {
				lines = append(lines, line)
				continue
			}
			entry := strings.Fields(line)
			if entry[0] != "127.0.0.1" {
				lines = append(lines, line)
				continue
			}
			if *add_hosts != "" {
				line = strings.Join(append(entry, *add_hosts), " ")
			}
			if *del_hosts != "" {
				tmp_entry := entry
				for i, v := range tmp_entry {
					if v == *del_hosts {
						entry = append(entry[:i], entry[i+1:]...)
					}
				}
				line = strings.Join(entry, " ")
			}
			lines = append(lines, line)
		}

		content := []byte(strings.Join(lines, "\n") + "\n")
		if err := ioutil.WriteFile(HostFilePath, content, 0644); err != nil {
			panic(err)
		}

		// delete DNS
		switch runtime.GOOS {
		case "darwin":
			if err := exec.Command("/usr/bin/dscacheutil", "-flushcache").Run(); err != nil {
				log.Fatal(err)
			}
		}

		os.Exit(0)
	}

	apath, err := filepath.Abs(*path)
	if err != nil {
		panic(err)
	}
	log.Printf("Start Serving HTTP => directory: %s, port: %d", apath, *port)

	var server = http.StripPrefix("/", http.FileServer(http.Dir(*path)))
	http.Handle("/", server)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		panic(err)
	}
}
