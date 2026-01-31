package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

func run(command string) error {
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {

	table("BUILDER")

	if argument("path") == "" || argument("call") == "" {
		write(" - path=[PATH]  izlenecek dizin")
		write(" - call=[CALL]  dosya değişince çalıştırılacak komut")
		return
	}

	path := argument("path")
	call := argument("call")

	
	write("├─── PATH: ", path);
	write("├─── CALL: ", call);
	

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Sadece dosya yazma / oluşturma olaylarında çalıştır
				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					write("Changed:", event.Name)

					err := run(call)
					if err != nil {
						write("Command error:", err)
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				write("Watcher error:", err)
			}
		}
	}()

	// Klasörleri recursive izle
	err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			watcher.Add(p)
			write("Watching:", p)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	<-done
}

/* -------------------- UTILS -------------------- */

func argument(key string, defaults ...string) string {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, key+"=") {
			val := strings.SplitN(arg, "=", 2)[1]
			return strings.Trim(val, "\"")
		}
	}
	if len(defaults) > 0 {
		return defaults[0]
	}
	return ""
}

func write(values ...interface{}) {
	originalFlags := log.Flags()
	log.SetFlags(0)
	log.Println(values...)
	log.SetFlags(originalFlags)
}

func center(text string) string {
	const width = 48
	if len(text) >= width {
		return text
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) +
		text +
		strings.Repeat(" ", width-len(text)-padding)
}

func table(name string) {
	write("╔════════════════════════════════════════════════╗")
	write("║" + center(name) + "║")
	write("╚════════════════════════════════════════════════╝")
}

func row(name string) {
	write("┌────────────────────────────────────────────────┐")
	write("│" + center(name) + "│")
	write("└────────────────────────────────────────────────┘")
}
