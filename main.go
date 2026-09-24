package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/process"
	"github.com/stretchr/testify/assert/yaml"
)

type ServerInfo struct {
	Name   string `yaml:"name"`
	Method string `yaml:"method"`
	Unit   string `yaml:"unit"`
	Match  string `yaml:"match"`
	Port   string `yaml:"port"`
}

type Config struct {
	ServerName        string       `yaml:"server-name"`
	ServerImage       string       `yaml:"server-image"`
	CorsHeader        string       `yaml:"corsheader"`
	InternalAuth      string       `yaml:"internalauth"`
	Port              string       `yaml:"port"`
	RefreshInterval   string       `yaml:"refresh-interval"`
	ReturnOnlyRunning bool         `yaml:"return-only-running"`
	Servers           []ServerInfo `yaml:"servers"`
}

var cfg Config

type ServerStatus struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
}

type ServerReport struct {
	ServerName   string         `json:"server_name"`
	ServerImage  string         `json:"server_image"`
	RunningCount int            `json:"running_count"`
	TotalCount   int            `json:"total_count"`
	Servers      []ServerStatus `json:"servers"`
}

type ServerCache struct {
	mu     sync.RWMutex
	report ServerReport
}

var cache = &ServerCache{}

func (c *ServerCache) Set(report ServerReport) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.report = report
}

func (c *ServerCache) Get() ServerReport {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.report
}

func startCacheRefresher(interval time.Duration) {
	cache.Set(fetchServerInformation())

	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			cache.Set(fetchServerInformation())
		}
	}()
}

func fetchServerInformation() ServerReport {
	var results []ServerStatus

	for _, s := range cfg.Servers {
		var isFound bool

		switch s.Method {
		case "systemd":
			cmd := exec.Command("systemctl", "is-active", s.Unit)
			output, err := cmd.Output()
			status := strings.TrimSpace(string(output))
			isFound = status == "active"
			if err != nil && status == "" {
				fmt.Printf("'%s' failed check status with err: %v\n", s.Name, err)
			}

		case "process":
			absolutePath, err := filepath.Abs(s.Match)
			if err != nil {
				fmt.Printf("'%s' failed check status with err: %v\n", s.Name, err)
				break
			}

			procs, err := process.Processes()
			if err != nil {
				fmt.Printf("'%s' failed check status with err: %v\n", s.Name, err)
				break
			}

			for _, p := range procs {
				cwd, err := p.Cwd()
				if err != nil {
					continue
				}
				exe, _ := p.Exe()

				if cwd == filepath.Dir(absolutePath) || exe == absolutePath {
					isFound = true
					break
				}
			}

		case "port":
			ln, err := net.Listen("tcp", "127.0.0.1:"+s.Port)
			if err != nil {
				isFound = true
			} else {
				fmt.Printf("'%s' failed check status with err: %v\n", s.Name, "port is free")
				ln.Close()
				isFound = false
			}
		}

		if !cfg.ReturnOnlyRunning || isFound {
			results = append(results, ServerStatus{Name: s.Name, Running: isFound})
		}
	}

	runningCount := 0
	for _, r := range results {
		if r.Running {
			runningCount++
		}
	}

	return ServerReport{
		ServerName:   cfg.ServerName,
		ServerImage:  cfg.ServerImage,
		RunningCount: runningCount,
		TotalCount:   len(results),
		Servers:      results,
	}
}

func main() {
	/* load & read config file */
	data, err := os.ReadFile("settings.yaml")
	if err != nil {
		panic(err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		panic(err)
	}

	/* cache */
	refreshTime, err := time.ParseDuration(cfg.RefreshInterval)
	if err != nil {
		panic(err)
	}
	startCacheRefresher(refreshTime)

	/* load endpoint */
	corsHeader := cfg.CorsHeader
	if corsHeader == "" {
		fmt.Printf("warning: corsheader in config file has not been properly provided, defaulting to public!\n")
		corsHeader = "*"
	}

	stats := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", corsHeader)
		w.Header().Set("Content-Type", "application/json")

		if cfg.InternalAuth == "" {
			fmt.Printf("warning: internal auth key has not been set, defaulting to not checking for header auth!\n")
		}

		if (cfg.InternalAuth != "") && (r.Header.Get("X-Internal-Auth") != cfg.InternalAuth) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		report := cache.Get()
		w.Header().Set("Content-Type", "application/json")

		jsonBytes, err := json.Marshal(report)
		if err != nil {
			http.Error(w, "failed to encode server report", http.StatusInternalServerError)
			return
		}

		w.Write(jsonBytes)
	}
	http.HandleFunc("/api/running-servers", stats)
	fmt.Printf("started reze-tracker service on port :%s\n", cfg.Port)
	http.ListenAndServe(":"+cfg.Port, nil)
}
