package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
)

func main() {
	r, _ := regexp.Compile("^(sd[a-z]+|nvme[0-9]+n[0-9]+)$")
	files, err := getDevices("/dev", r)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v\n", files)
}

func getDevices(dir string, r *regexp.Regexp) (names []string, err error) {
	files, err := os.ReadDir("/dev")
	if err != nil {
		return names, err
	}

	for _, file := range files {
		if r.MatchString(file.Name()) {
			names = append(names, file.Name())
		}
	}
	return names, err
}
