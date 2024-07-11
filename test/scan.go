package main

import (
	"bufio"
	"log"
	"os"
	"time"
)

func main() {
	watchEveLogFile()
}

func watchEveLogFile() {
	for {
		// Open the eve.json log file
		file, err := os.Open("scan.txt")
		if err != nil {
			if os.IsNotExist(err) {
				log.Println("eve.json does not exist, waiting...")
			} else {
				log.Fatalf("Failed to open eve.json: %v", err)
			}
			time.Sleep(10 * time.Second)
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			log.Printf("line:%s\n", string(scanner.Bytes()))
		}

		if err := scanner.Err(); err != nil {
			log.Printf("Error reading eve.json: %v", err)
		}

		file.Close()
		time.Sleep(time.Second)
	}
}
