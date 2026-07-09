package main

import(
  "sync/atomic"
  "time"
  "sync"
  "os"
)


var CurrentTime atomic.Value

func Init_LogTime() {
	go func() {
		ticker := time.NewTicker(time.Second)

		for {
			CurrentTime.Store(time.Now().Format("2006-01-02 15:04:05"))
			<-ticker.C
		}
	}()
}

func Log_Now() string {
	return CurrentTime.Load().(string)
}



// LOG Sistemi
var (
	LOG_CHAN chan string
	LOG_MU   sync.Mutex
	STOP_LOG chan struct{}
)

func Init_Log() {
	LOG_MU.Lock()
	defer LOG_MU.Unlock()

	// önce eski logger'ı durdur
	if STOP_LOG != nil {
		close(STOP_LOG)
	}

	LOG_CHAN = make(chan string, 1024)
	STOP_LOG = make(chan struct{})

	go func(ch chan string, stop chan struct{}) {
		f, err := os.OpenFile(server.Log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			write("[X] Log File Doesnt Readed", server.Log)
			return
		}
		defer f.Close()

		for {
			select {
			case msg := <-ch:
				if _, err := f.WriteString(msg + "\n"); err != nil {
					// disk full vs.
				}
			case <-stop:
				return
			}
		}
	}(LOG_CHAN, STOP_LOG)
}

func Log(msg string) {
	LOG_MU.Lock()
	ch := LOG_CHAN
	LOG_MU.Unlock()

	if ch == nil {
		return
	}

	select {
	case ch <- msg:
	default:
		// doluysa düşür
	}
}
