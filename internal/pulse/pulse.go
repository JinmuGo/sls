// Package pulse provides a lightweight, non-blocking telemetry client
// for the Pulse event collection server (pulse.jinmu.me).
//
// Usage:
//
//	pulse.Init("0.4.2")           // call once at startup
//	pulse.Track("command_run", pulse.Props{"command": "connect"})
//	defer pulse.Shutdown()        // flush remaining events on exit
//
// Telemetry is opt-in: on first run, the user is prompted to consent.
// Consent is stored in ~/.config/sls/telemetry.
// Can also be controlled via PULSE_DISABLED=1 or SLS_TELEMETRY=off.
// Events are queued in memory and flushed asynchronously.
// Network failures are silently ignored.
package pulse

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	defaultEndpoint  = "https://pulse.jinmu.me"
	defaultAPIKey    = "pulse-dev-key"
	flushInterval    = 30 * time.Second
	maxQueueSize     = 50
	batchThreshold   = 10
	httpTimeout      = 2 * time.Second
	consentFile      = "telemetry" // stored in ~/.config/sls/
)

// Props is a shorthand for event properties.
type Props map[string]interface{}

type event struct {
	App        string                 `json:"app"`
	Event      string                 `json:"event"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Timestamp  string                 `json:"timestamp"`
}

var (
	mu       sync.Mutex
	queue    []event
	ctx      map[string]interface{}
	client   *http.Client
	endpoint string
	apiKey   string
	disabled bool
	started  bool
	stopCh   chan struct{}
	wg       sync.WaitGroup
)

// Init initializes the telemetry client with the given sls version.
// Must be called once before Track(). Safe to call even if telemetry is disabled.
func Init(version string) {
	if isDisabled() || !hasConsent() {
		disabled = true
		return
	}

	endpoint = getEnv("PULSE_ENDPOINT", defaultEndpoint)
	apiKey = getEnv("PULSE_API_KEY", defaultAPIKey)

	ctx = map[string]interface{}{
		"os":      runtime.GOOS,
		"arch":    runtime.GOARCH,
		"version": version,
		"runtime": fmt.Sprintf("go%s", runtime.Version()[2:]), // strip "go" prefix from "go1.25.0"
	}

	client = &http.Client{Timeout: httpTimeout}
	queue = make([]event, 0, maxQueueSize)
	stopCh = make(chan struct{})
	started = true

	wg.Add(1)
	go flushLoop()
}

// Track queues a telemetry event. Non-blocking, safe to call from any goroutine.
// Does nothing if telemetry is disabled or Init() hasn't been called.
func Track(eventName string, properties Props) {
	if disabled || !started {
		return
	}

	e := event{
		App:        "sls",
		Event:      eventName,
		Context:    ctx,
		Properties: properties,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	mu.Lock()
	if len(queue) >= maxQueueSize {
		// Drop oldest event
		queue = queue[1:]
	}
	queue = append(queue, e)
	shouldFlush := len(queue) >= batchThreshold
	mu.Unlock()

	if shouldFlush {
		go flush()
	}
}

// Shutdown flushes any remaining events and stops the background goroutine.
// Call with defer in main(). Has a 2-second timeout.
func Shutdown() {
	if disabled || !started {
		return
	}

	close(stopCh)
	flush()
	wg.Wait()
}

func flushLoop() {
	defer wg.Done()
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			flush()
		case <-stopCh:
			return
		}
	}
}

func flush() {
	mu.Lock()
	if len(queue) == 0 {
		mu.Unlock()
		return
	}
	batch := make([]event, len(queue))
	copy(batch, queue)
	queue = queue[:0]
	mu.Unlock()

	body := map[string]interface{}{
		"events": batch,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return // silently drop
	}

	req, err := http.NewRequest("POST", endpoint+"/v1/events/batch", bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return // network failure, silently drop
	}
	resp.Body.Close()
}

func isDisabled() bool {
	if os.Getenv("PULSE_DISABLED") == "1" {
		return true
	}
	if os.Getenv("SLS_TELEMETRY") == "off" {
		return true
	}
	return false
}

func consentPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "sls", consentFile)
}

// hasConsent checks whether the user has opted in to telemetry.
// If no consent file exists, prompts the user interactively.
func hasConsent() bool {
	path := consentPath()
	if path == "" {
		return false
	}

	data, err := os.ReadFile(path)
	if err == nil {
		return strings.TrimSpace(string(data)) == "enabled"
	}

	// No consent file — prompt the user
	return promptConsent(path)
}

func promptConsent(path string) bool {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  sls would like to collect anonymous usage data to improve the tool.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  What we collect:")
	fmt.Fprintln(os.Stderr, "    - Command name (e.g. connect, scan)")
	fmt.Fprintln(os.Stderr, "    - OS, architecture, sls version")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  What we DON'T collect:")
	fmt.Fprintln(os.Stderr, "    - Hostnames, IP addresses, or SSH config")
	fmt.Fprintln(os.Stderr, "    - Container names or any personal information")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  You can change this later: export PULSE_DISABLED=1")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprint(os.Stderr, "  Enable telemetry? [y/N] ")

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	consented := answer == "y" || answer == "yes"

	value := "disabled"
	if consented {
		value = "enabled"
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(value+"\n"), 0o644)

	return consented
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
