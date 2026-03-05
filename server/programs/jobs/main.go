package main

import (
	"log"
	"os"
	"os/exec"
	"time"
)

var logger *log.Logger

func runJob() {
	cmd := exec.Command("/bin/bash", "/web/jobs/every-minute.sh")

	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Printf("job error: %v", err)
	}

	if len(output) > 0 {
		logger.Printf("job output:\n%s", output)
	}
}

func main() {
	logFile, err := os.OpenFile(
		"/web/jobs/every-minute.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		log.Fatalf("log file açılamadı: %v", err)
	}
	defer logFile.Close()

	logger = log.New(logFile, "", log.LstdFlags)

	logger.Println("Scheduler started (runs every minute)")

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C
		runJob()
	}
}
