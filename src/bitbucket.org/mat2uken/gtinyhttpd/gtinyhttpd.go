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
	var http_port *int = flag.Int("http_port", 8080, "serving port for http.")
	var https_port *int = flag.Int("https_port", 8443, "serving port for http.")
	var ssl_hostname *string = flag.String("ssl-host", "", "https hostname.")
	var ssl_cert_file_path *string = flag.String("ssl-cert", "", "ssl certificate file(including chain cert).")
	var ssl_key_file_path *string = flag.String("ssl-key", "", "ssl certificate key file.")
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

			// skip: comment out line
			if strings.Index(line, "#") == 0 {
				lines = append(lines, line)
				continue
			}

			// split by whitespaces
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

		// clear DNS Cache by OS layer if exists
		switch runtime.GOOS {
		case "darwin":
			if err := exec.Command("/usr/bin/dscacheutil", "-flushcache").Run(); err != nil {
				log.Fatalf("cannnot run command: dscacheutil -flushcache => %v", err)
			}
			//			out, oerr := exec.Command("/usr/bin/killall", "-HUP", "mDNSResponder").Output()
			//			log.Printf("out: %v, err:%v", out, oerr)
			if err := exec.Command("/usr/bin/killall", "-HUP", "mDNSResponder").Run(); err != nil {
				log.Fatalf("cannot run command: killall -HUP mDNSResponder => %v", err)
			}
		}

		os.Exit(0)
	}

	apath, err := filepath.Abs(*path)
	if err != nil {
		panic(err)
	}
	var server = http.StripPrefix("/", http.FileServer(http.Dir(*path)))
	http.Handle("/", server)

	go func() {
		log.Printf("Start Serving HTTP => directory: %s, http_port: %d", apath, *http_port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", *http_port), nil); err != nil {
			log.Fatalf("cannot listen http: port=>%d", *http_port)
			panic(err)
		}
	}()

	if *ssl_cert_file_path != "" && *ssl_key_file_path != "" {
		log.Printf("Start Serving HTTPS => directory: %s, https_port: %d", apath, *https_port)
		go func() {
			if err := http.ListenAndServeTLS(fmt.Sprintf(":%d", *https_port), *ssl_cert_file_path, *ssl_key_file_path, nil); err != nil {
				log.Fatalf("cannot listen https: port=>%d", *https_port)
				panic(err)
			}
		}()
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		log.Printf("input line: %v", line)
	}
}
