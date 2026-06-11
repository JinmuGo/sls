// Package pulse provides a lightweight, non-blocking telemetry client that ships
// anonymous usage events to the self-hosted observability stack's OTLP ingest
// front door (ingest.jinmu.me). Events land in VictoriaLogs and surface in the
// portfolio Grafana under the service label "sls".
//
// Usage:
//
//	pulse.Init("0.4.2")           // call once at startup
//	pulse.Track("command_run", pulse.Props{"command": "connect"})
//	defer pulse.Shutdown()        // flush remaining events on exit
//
// Wire format is OTLP/HTTP logs (POST /v1/logs, Authorization: Bearer <token>).
// Each event becomes one OTLP log record: resource attribute service.name="sls",
// the record body is the event name, and the flat record attributes (event,
// anon_id, os, arch, version, runtime, plus any Props) are what the Grafana
// panels key on — count_uniq(anon_id) for active users, stats by(event) for top
// events. The collector copies service.name -> the "service" field used by the
// dashboard's $service variable.
//
// Telemetry is opt-in: on first run, the user is prompted to consent. Consent is
// stored in ~/.config/sls/telemetry. Can also be controlled via PULSE_DISABLED=1
// or SLS_TELEMETRY=off. Events are queued in memory and flushed asynchronously;
// network failures are silently ignored. The only identifiers that ever leave the
// process are the event/command names, OS/arch/version, and a random per-install
// anon_id (a v4 UUID in ~/.config/sls/anon_id, derived from nothing — never a
// hostname, IP, path, or SSH config).
package pulse

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Overridable at build time via -ldflags "-X github.com/jinmugo/sls/internal/pulse.defaultEndpoint=... -X ...defaultIngestToken=...".
// defaultIngestToken is the Bearer token the public CLI presents to the ingest
// front door; it is a separately-revocable token (NOT the relay's server token).
var (
	defaultEndpoint    = "https://ingest.jinmu.me"
	defaultIngestToken = "ingest-dev-token"
)

const (
	flushInterval  = 30 * time.Second
	maxQueueSize   = 50
	batchThreshold = 10
	httpTimeout    = 2 * time.Second
	consentFile    = "telemetry" // stored in ~/.config/sls/
	anonIDFile     = "anon_id"   // stored in ~/.config/sls/
	serviceName    = "sls"       // OTLP resource attribute service.name
)

// Props is a shorthand for event properties. Values are strings because OTLP log
// attributes are emitted as stringValue and the project bans the any type.
type Props map[string]string

// event is the in-memory representation queued between flushes. attrs is the
// pre-built flat OTLP attribute list (event, anon_id, context, and Props); nano
// is the capture time in Unix nanoseconds.
type event struct {
	name  string
	nano  int64
	attrs []otlpKeyValue
}

var (
	mu        sync.Mutex
	queue     []event
	ctxAttrs  []otlpKeyValue // os/arch/version/runtime, built once in Init
	anonID    string
	client    *http.Client
	endpoint  string
	token     string
	userAgent string
	disabled  bool
	started   bool
	stopCh    chan struct{}
	wg        sync.WaitGroup
)

// ---- OTLP/HTTP logs payload (typed; the project bans the any type) ----------

type otlpStringValue struct {
	StringValue string `json:"stringValue"`
}

type otlpKeyValue struct {
	Key   string          `json:"key"`
	Value otlpStringValue `json:"value"`
}

type otlpLogRecord struct {
	TimeUnixNano string          `json:"timeUnixNano"`
	Body         otlpStringValue `json:"body"`
	Attributes   []otlpKeyValue  `json:"attributes"`
}

type otlpScopeLogs struct {
	LogRecords []otlpLogRecord `json:"logRecords"`
}

type otlpResource struct {
	Attributes []otlpKeyValue `json:"attributes"`
}

type otlpResourceLogs struct {
	Resource  otlpResource    `json:"resource"`
	ScopeLogs []otlpScopeLogs `json:"scopeLogs"`
}

type otlpLogsBody struct {
	ResourceLogs []otlpResourceLogs `json:"resourceLogs"`
}

// kv builds a flat OTLP string attribute.
func kv(key, val string) otlpKeyValue {
	return otlpKeyValue{Key: key, Value: otlpStringValue{StringValue: val}}
}

// Init initializes the telemetry client with the given sls version.
// Must be called once before Track(). Safe to call even if telemetry is disabled.
func Init(version string) {
	if isDisabled() || !hasConsent() {
		disabled = true
		return
	}

	endpoint = strings.TrimRight(getEnv("TELEMETRY_ENDPOINT", defaultEndpoint), "/")
	token = getEnv("TELEMETRY_API_KEY", defaultIngestToken)
	userAgent = "sls/" + version

	ctxAttrs = []otlpKeyValue{
		kv("os", runtime.GOOS),
		kv("arch", runtime.GOARCH),
		kv("version", version),
		kv("runtime", fmt.Sprintf("go%s", runtime.Version()[2:])), // strip "go" prefix from "go1.25.0"
	}
	anonID = loadOrCreateAnonID()

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

	// Pre-build the flat OTLP attributes the Grafana panels key on exactly:
	// "event" + "anon_id" are required (stats by(event), count_uniq(anon_id)).
	attrs := make([]otlpKeyValue, 0, 2+len(ctxAttrs)+len(properties))
	attrs = append(attrs, kv("event", eventName), kv("anon_id", anonID))
	attrs = append(attrs, ctxAttrs...)
	for k, v := range properties {
		attrs = append(attrs, kv(k, v))
	}

	e := event{
		name:  eventName,
		nano:  time.Now().UnixNano(),
		attrs: attrs,
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

// flush drains the queue and POSTs it as a single OTLP/HTTP logs request: one
// resourceLogs (service.name="sls") carrying every queued event as a log record.
// All errors are swallowed — telemetry must never affect the main flow.
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

	records := make([]otlpLogRecord, len(batch))
	for i, e := range batch {
		records[i] = otlpLogRecord{
			TimeUnixNano: strconv.FormatInt(e.nano, 10),
			Body:         otlpStringValue{StringValue: e.name},
			Attributes:   e.attrs,
		}
	}
	payload := otlpLogsBody{
		ResourceLogs: []otlpResourceLogs{{
			Resource:  otlpResource{Attributes: []otlpKeyValue{kv("service.name", serviceName)}},
			ScopeLogs: []otlpScopeLogs{{LogRecords: records}},
		}},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return // silently drop
	}

	req, err := http.NewRequest("POST", endpoint+"/v1/logs", bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	// Identify as a named client. Go's default User-Agent (Go-http-client/1.1)
	// trips Cloudflare's bot-mitigation challenge on the ingest edge (HTTP 403);
	// a non-browser client UA passes it.
	req.Header.Set("User-Agent", userAgent)

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

	// No consent file. Only prompt when running interactively — otherwise (piped
	// stdin, CI, shell-completion generation, scripted `sls connect`) reading
	// stdin would block or consume unrelated input, and we'd persist a decision
	// the user never saw. Treat non-interactive as "no consent" without writing
	// the file, so a later interactive run can still ask.
	if !isInteractive() {
		return false
	}
	return promptConsent(path)
}

// isInteractive reports whether stdin is connected to a terminal.
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
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

// loadOrCreateAnonID returns a stable, random per-install identifier used for
// count_uniq(anon_id) active-user counts. It is read from ~/.config/sls/anon_id,
// generating and persisting a fresh v4 UUID on first use. The id is derived from
// crypto/rand alone — it encodes no hostname, user, or machine fingerprint, so it
// cannot be reversed to its source. Returns "" if the home dir is unavailable.
func loadOrCreateAnonID() string {
	path := anonIDPath()
	if path == "" {
		return ""
	}
	if data, err := os.ReadFile(path); err == nil {
		if s := strings.TrimSpace(string(data)); s != "" {
			return s
		}
	}
	id := newAnonID()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(id+"\n"), 0o600)
	return id
}

func anonIDPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "sls", anonIDFile)
}

// newAnonID returns a random RFC 4122 v4 UUID string.
func newAnonID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fall back to a time-based, still non-PII token if the RNG is unavailable.
		return "t-" + strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
