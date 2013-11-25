package main

import "net/http"
import "flag"
import "log"
import "path/filepath"
import "fmt"
import "os"
import "os/exec"
import "os/signal"
import "io/ioutil"
import "bufio"
import "strings"
import "runtime"

const (
	//	HostFilePath = "/Users/ku/Desktop/gtinyhttpd/hosts"
	HostFilePath = "/private/etc/hosts"
)

type EditHostFileHandlerFunc func(entry []string) []string

func EditHostsFile(edit_func EditHostFileHandlerFunc) {
	var lines []string
	f, err := os.Open(HostFilePath)
	if err != nil {
		panic(err)
	}
	r := bufio.NewScanner(f)
	for r.Scan() {
		line := r.Text()

		// split by whitespaces
		entry := strings.Fields(line)

		if strings.Index(entry[0], "#") == 0 {
			lines = append(lines, line)
			continue
		}
		if entry[0] != "127.0.0.1" {
			lines = append(lines, line)
			continue
		}

		lines = append(lines, strings.Join(edit_func(entry), " "))
	}

	content := []byte(strings.Join(lines, "\n") + "\n")
	if err := ioutil.WriteFile(HostFilePath, content, 0644); err != nil {
		panic(err)
	}
}

func AddLocalHostNameToHostsFile(hostname string) {
	edit_func := func(entry []string) []string {
		for _, v := range entry {
			if v == hostname {
				return entry
			}
		}
		return append(entry, hostname)
	}
	EditHostsFile(edit_func)
}

func RemoveLocalHostNameFromHostsFile(hostname string) {
	edit_func := func(entry []string) []string {
		tmp_entry := entry
		for i, v := range tmp_entry {
			if v == hostname {
				entry = append(entry[:i], entry[i+1:]...)
			}
		}
		return entry
	}
	EditHostsFile(edit_func)
}

func ClearDNSCache() {
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
}

func main() {
	var add_hostname *string = flag.String("add-hosts", "", "add entry to hosts.")
	var del_hostname *string = flag.String("del-hosts", "", "del entry to hosts.")
	var path *string = flag.String("path", ".", "serving files dir path.")
	var http_port *int = flag.Int("http_port", 8080, "serving port for http.")
	var https_port *int = flag.Int("https_port", 8443, "serving port for http.")
	var ssl_hostname *string = flag.String("ssl-host", "", "https hostname.")
	var ssl_cert_file_path *string = flag.String("ssl-cert", "", "ssl certificate file(including chain cert).")
	var ssl_key_file_path *string = flag.String("ssl-key", "", "ssl certificate key file.")
	flag.Parse()

	if *add_hostname != "" || *del_hostname != "" {
		uid := os.Getuid()
		if uid != 0 {
			log.Printf("are you root? => uid: %d", uid)
			os.Exit(1)
		}

		if *add_hostname != "" {
			AddLocalHostNameToHostsFile(*add_hostname)
		}
		if *del_hostname != "" {
			RemoveLocalHostNameFromHostsFile(*del_hostname)
		}

		ClearDNSCache()

		os.Exit(0)
	}

	apath, err := filepath.Abs(*path)
	if err != nil {
		panic(err)
	}
	var server = http.StripPrefix("/", http.FileServer(http.Dir(*path)))
	http.Handle("/", server)

	log.Printf("Start Serving HTTP => directory: %s, http_port: %d", apath, *http_port)
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", *http_port), nil); err != nil {
			log.Fatalf("cannot listen http: port=>%d", *http_port)
			return
		}
	}()

	if *ssl_cert_file_path != "" && *ssl_key_file_path != "" {
		uid := os.Getuid()
		if uid != 0 {
			log.Printf("Cannot run HTTPS server. You must run as root.")
			log.Printf("are you root? => uid: %d", uid)
			os.Exit(1)
		}

		if *ssl_hostname == "" {
			log.Fatal("You must specify hostname to enable ssl certificate.")
			os.Exit(1)
		}

		log.Printf("Adding ssl-hostname to hosts file: hostname=>%v", *ssl_hostname)
		AddLocalHostNameToHostsFile(*ssl_hostname)
		ClearDNSCache()

		log.Printf("Start Serving HTTPS => directory: %s, https_port: %d", apath, *https_port)
		go func() {
			if err := http.ListenAndServeTLS(fmt.Sprintf(":%d", *https_port),
				*ssl_cert_file_path, *ssl_key_file_path, nil); err != nil {
				log.Fatalf("cannot listen https: port=>%d", *https_port)
				RemoveLocalHostNameFromHostsFile(*ssl_hostname)
				return
			}
		}()

		go func() {
			c := make(chan os.Signal)
			signal.Notify(c, os.Interrupt)
			s := <-c
			RemoveLocalHostNameFromHostsFile(*ssl_hostname)
			log.Printf("Removed hosts file entry: ssl_hostname => %v", *ssl_hostname)

			log.Printf("Exiting with %v", s)
			os.Exit(0)
		}()
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		log.Printf("line: %v", line)
		switch line[:1] {
		case "a":
			AddLocalHostNameToHostsFile(*ssl_hostname)
			log.Printf("Added hosts file entry: ssl_hostname => %v", *ssl_hostname)
		case "r":
			RemoveLocalHostNameFromHostsFile(*ssl_hostname)
			log.Printf("Removed hosts file entry: ssl_hostname => %v", *ssl_hostname)
		case "q":
			RemoveLocalHostNameFromHostsFile(*ssl_hostname)
			log.Printf("Removed hosts file entry: ssl_hostname => %v", *ssl_hostname)
			log.Printf("Exiting by key 'q")
			os.Exit(0)
		}
	}
}
