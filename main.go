package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
)

type Session struct {
	Folder       string    `json:"folder"`
	StartedAt    time.Time `json:"started_at"`
	LastUpdate   time.Time `json:"last_update"`
	Duration     string    `json:"duration"`
	ChangedPaths []string  `json:"changed_paths"`
}

var (
	activeSession Session
	sessionLock   sync.Mutex
)

const logInterval = 10 * time.Second

// Ekstensi bahasa pemrograman yang ingin dipantau
var allowedExtensions = map[string]bool{
	".go":   true,
	".js":   true,
	".ts":   true,
	".jsx":  true,
	".tsx":  true,
	".py":   true,
	".php":  true,
	".java": true,
	".rb":   true,
	".cpp":  true,
	".c":    true,
	".cs":   true,
	".html": true,
	".css":  true,
	".rs":   true,
	".dart": true,
	".kt":   true,
}

func main() {
	color.Cyan("🧠 CodeTimer Advanced Tracker")

	var folder string
	fmt.Print("📁 Masukkan folder yang ingin dipantau: ")
	fmt.Scanln(&folder)

	if _, err := os.Stat(folder); os.IsNotExist(err) {
		color.Red("❌ Folder tidak ditemukan!")
		return
	}

	activeSession = Session{
		Folder:       folder,
		StartedAt:    time.Now(),
		LastUpdate:   time.Now(),
		ChangedPaths: []string{},
	}

	go monitorFolder(folder)
	go autoSaveLog()

	// Handle Ctrl+C
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	color.Magenta("\n🛑 Program dihentikan...")
	saveSession(true)
}

func monitorFolder(folder string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		color.Red("⚠️ Gagal membuat watcher: %v", err)
		return
	}
	defer watcher.Close()

	filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			watcher.Add(path)
		}
		return nil
	})

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				ext := strings.ToLower(filepath.Ext(event.Name))
				if allowedExtensions[ext] {
					sessionLock.Lock()
					activeSession.LastUpdate = time.Now()
					if !containsPath(activeSession.ChangedPaths, event.Name) {
						activeSession.ChangedPaths = append(activeSession.ChangedPaths, event.Name)
					}
					sessionLock.Unlock()
					color.Yellow("🔧 File berubah: %s", event.Name)
				}
			}
		case err := <-watcher.Errors:
			color.Red("⚠️ Watcher error: %v", err)
		}
	}
}

func autoSaveLog() {
	for {
		time.Sleep(logInterval)
		sessionLock.Lock()
		saveSession(false)
		sessionLock.Unlock()
	}
}

func saveSession(final bool) {
	activeSession.Duration = time.Since(activeSession.StartedAt).String()

	err := os.MkdirAll("logs", os.ModePerm)
	if err != nil {
		color.Red("❌ Gagal membuat folder log: %v", err)
		return
	}

	timestamp := activeSession.StartedAt.Format("2006-01-02_15-04-05")
	filename := filepath.Join("logs", fmt.Sprintf("%s.json", timestamp))

	data, _ := json.MarshalIndent(activeSession, "", "  ")
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		color.Red("❌ Gagal menyimpan log: %v", err)
	} else if final {
		color.Green("📁 Log akhir disimpan: %s", filename)
		fmt.Printf("⏱️  Durasi sesi: %s\n", activeSession.Duration)
		fmt.Printf("🔄 Total file berubah: %d\n", len(activeSession.ChangedPaths))
		for _, path := range activeSession.ChangedPaths {
			fmt.Println(" 🔸", path)
		}
	}
}

func containsPath(paths []string, target string) bool {
	for _, p := range paths {
		if p == target {
			return true
		}
	}
	return false
}
