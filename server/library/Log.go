/*

USAGE:

var Logger *Logger = NewLogger(FILE_NAME)

func main() {
	Logger.Log("Program başladı")
	Logger.Log("Bir mesaj")
	defer Logger.Close()
}

*/
package main

import (
	"os"
	"sync"
)

type Logger struct {
	fileName string

	logChan chan string
	stop    chan struct{}

	mu sync.Mutex
}

func NewLogger(fileName string) *Logger {
	l := &Logger{
		fileName: fileName,
		logChan:  make(chan string, 1024),
		stop:     make(chan struct{}),
	}

	go l.run()

	return l
}

func (l *Logger) run() {
	f, err := os.OpenFile(l.fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		write("[X] Log File Doesn't Open:", l.fileName)
		return
	}else{
		write("[+] ",l.fileName+" ready to log")
	}
	defer f.Close()

	for {
		select {
		case msg := <-l.logChan:
			if _, err := f.WriteString(msg + "\n"); err != nil {
				// disk full vb.
			}

		case <-l.stop:
			return
		}
	}
}

func (l *Logger) Log(msg string) {
	l.mu.Lock()
	ch := l.logChan
	l.mu.Unlock()

	if ch == nil {
		return
	}

	select {
	case ch <- msg:
	default:
		// kanal doluysa log düşürülür
	}
}

func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.stop != nil {
		close(l.stop)
		l.stop = nil
	}
}
