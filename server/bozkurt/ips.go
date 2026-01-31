package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	LogFilePath = "/web/.log/topluyo.log"
	WindowSec   = 120
	Limit       = 100
)

type Hit struct {
	Time time.Time
}

var (
	ipHits  = make(map[string][]Hit)
	printed = make(map[string]bool)
)

func main() {
	fmt.Println("🐺 BozKurt started...")

	file, err := os.Open(LogFilePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// dosyanın SONUNA git (tail -F)
	file.Seek(0, os.SEEK_END)

	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		handleLine(strings.TrimSpace(line))
	}
}

func handleLine(line string) {
	if !strings.HasPrefix(line, "RATEOVER") {
		return
	}

	// RATEOVER, 2026-01-07 01:40:33, 49.0.32.50, domain, /
	parts := strings.Split(line, ",")
	if len(parts) < 3 {
		return
	}

	timestamp, err := time.Parse("2006-01-02 15:04:05", strings.TrimSpace(parts[1]))
	if err != nil {
		return
	}

	ip := strings.TrimSpace(parts[2])

	now := time.Now()
	windowStart := now.Add(-WindowSec * time.Second)

	// eski kayıtları temizle
	var recent []Hit
	for _, h := range ipHits[ip] {
		if h.Time.After(windowStart) {
			recent = append(recent, h)
		}
	}

	recent = append(recent, Hit{Time: timestamp})
	ipHits[ip] = recent

	if len(recent) >= Limit && !printed[ip] {
		fmt.Println(ip)
		printed[ip] = true
	}
}
