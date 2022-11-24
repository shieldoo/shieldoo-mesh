package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
)

// readLines reads a whole file into memory
// and returns a slice of its lines.
func readLines(path string, filterout string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		l := scanner.Text()
		if !strings.HasSuffix(l, filterout) {
			lines = append(lines, l)
		}
	}
	return lines, scanner.Err()
}

// writeLines writes the lines to the given file.
func writeLines(lines []string, path string) error {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}

// if the error happens it is not critical for us, we are only showing log message
func loadDNS() {
	log.Debug("loadDNS() - loading ...")
	path := "/etc/hosts"
	if runtime.GOOS == "windows" {
		path = os.Getenv("SystemRoot") + `\System32\drivers\etc\hosts`
	}

	filterout := "# XPIGZRUZDKFSITQS-nebula custom records"
	// load file to array (without old records)
	hosts, err := readLines(path, filterout)
	if err != nil {
		log.Error("error loading hosts file: ", err)
		return
	}

	for _, v := range dnsconf.DnsRecords {
		hosts = append(hosts, v+" "+filterout)
	}

	err = writeLines(hosts, path)
	if err != nil {
		log.Error("error writing hosts file: ", err)
		return
	}
}
