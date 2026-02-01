package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func printLogo() {
	logo := `
  _____                      ______             
 |  __ \                    |  ____|            
 | |__) | __ _____  ___   _ | |__  _   _  ___   
 |  ___/ '__/ _ \ \/ / | | ||  __|| | | |/ _ \  
 | |   | | | (_) >  <| |_| || |___| |_| |  __/  
 |_|   |_|  \___/_/\_\\__, ||______\__, |\___|  
                       __/ |        __/ |       
                      |___/        |___/        `

	// Print in Cyan using ANSI colors
	fmt.Printf("\033[36m%s\033[0m\n", logo)
	fmt.Println(" =============================================")
}

//go:embed index.html
var staticFiles embed.FS

// Define a custom type for context keys to avoid collisions
type key string

const startTimeKey key = "startTime"

var (
	upgrader  = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	clients   = make(map[*websocket.Conn]bool)
	clientsMu sync.Mutex
	broadcast = make(chan CombinedLog)
	// Create a separate channel for CLI
	cliChan = make(chan CombinedLog)
)

type CombinedLog struct {
	Method      string `json:"method"`
	QueryString string `json:"query_string"`
	Path        string `json:"path"`
	ReqHeaders  string `json:"req_headers"`
	Status      int    `json:"status"`
	ReqBody     string `json:"req_body"`
	RespHeaders string `json:"resp_headers"`
	RespBody    string `json:"resp_body"`
	Latency     string `json:"latency"`
	Time        string `json:"time"`
}

var (
	history      []CombinedLog
	historyMutex sync.Mutex
	maxHistory   = 50
)

func main() {
	printLogo()
	uiPort := flag.String("ui", "4040", "port for the inspector UI")
	portPtr := flag.String("p", "3000", "target port to proxy")
	domainPtr := flag.String("domain", "localhost", "custom domain name")
	flag.Parse()
	// Get the port from the argument if provided (e.g., ./proxyeye 8080)
	targetPort := *portPtr
	customDomain := *domainPtr
	args := flag.Args()
	if len(args) > 0 {
		targetPort = args[0]
	}

	// 2. Build the target URL dynamically
	targetURL := fmt.Sprintf("http://127.0.0.1:%s", targetPort)
	uiAddr := ":" + *uiPort
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatal("Invalid target port")
	}
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Intercept the Response
	proxy.ModifyResponse = func(r *http.Response) error {
		// 1. Capture the headers IMMEDIATELY
		// We clone them because the proxy might mutate 'r' later
		capturedHeaders := make(http.Header)
		for k, v := range r.Header {
			capturedHeaders[k] = v
		}

		dump, _ := httputil.DumpResponse(r, false)
		dumpRequest, _ := httputil.DumpRequest(r.Request, false)
		// 2. Standard body processing
		resBody, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(resBody)) // Reset for client

		var latency string
		if startTime, ok := r.Request.Context().Value(startTimeKey).(time.Time); ok {
			// Convert to milliseconds and format to 2 decimal places
			ms := float64(time.Since(startTime)) / 1e6
			latency = fmt.Sprintf("%.2fms", ms)
		}
		ctx := r.Request.Context()
		reqBody, _ := ctx.Value("capturedReqBody").(string)

		broadcast <- CombinedLog{
			Method:      r.Request.Method,
			Path:        r.Request.URL.Path,
			QueryString: r.Request.URL.RawQuery,
			ReqHeaders:  string(dumpRequest),
			Status:      r.StatusCode,
			ReqBody:     reqBody,
			RespHeaders: string(dump),
			RespBody:    string(resBody),
			Latency:     latency,
			Time:        time.Now().Format("15:04:05"),
		}
		return nil
	}

	// 1. WebSocket Route
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, _ := upgrader.Upgrade(w, r, nil)
		clientsMu.Lock()
		clients[ws] = true
		clientsMu.Unlock()
	})

	// 2. Proxy + Request Timer
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws" ||
			r.URL.Path == "/inspect" ||
			r.URL.Path == "/favicon.ico" ||
			strings.Contains(r.URL.Path, ".well-known") {
			return
		}
		// Inject start time into context
		// --- Intercept Request Body ---
		var reqBodyBytes []byte
		if r.Body != nil {
			reqBodyBytes, _ = io.ReadAll(r.Body)
		}
		// Restore the body so the proxy can still send it to the target
		r.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))

		ctx := r.Context()
		start := time.Now()
		ctx = context.WithValue(r.Context(), startTimeKey, start)
		ctx = context.WithValue(ctx, "capturedReqBody", string(reqBodyBytes))
		r = r.WithContext(ctx)
		proxy.ServeHTTP(w, r)
	})

	http.HandleFunc("/inspect", func(w http.ResponseWriter, r *http.Request) {
		data, _ := staticFiles.ReadFile("index.html")
		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
	})

	http.HandleFunc("/history", func(w http.ResponseWriter, r *http.Request) {
		historyMutex.Lock()
		defer historyMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(history)
	})

	go handleBroadcasts()                                     // For Web UI
	go startCLIDashboard(targetPort, targetURL, customDomain) // For Terminal UI

	fmt.Printf("🚀 ProxyEye: http://localhost:%s/inspect\n", *uiPort)
	fmt.Printf("🚀 Proxying: http://localhost:%s -> %s\n", *uiPort, targetURL)
	log.Fatal(http.ListenAndServe(uiAddr, nil))
}

func handleBroadcasts() {
	for {
		// Grab the next log from the channel
		msg := <-broadcast
		// Send to CLI channel
		cliChan <- msg
		saveToHistory(msg)

		// Send it to every connected client
		clientsMu.Lock()
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				//log.Printf("Websocket error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
		clientsMu.Unlock()
	}
}

func startCLIDashboard(target, targetURL, customDomain string) {
	// Clear screen and print static header once
	fmt.Print("\033[H\033[2J")
	fmt.Printf("Session: online\n")
	fmt.Printf("Domain: %s | Forwarding: %s\n\n", customDomain, targetURL)
	fmt.Println("\nHTTP Requests")
	fmt.Println("-------------")

	for {
		msg := <-cliChan // Read from dedicated CLI channel

		// Color logic: Green for success, Red for errors
		color := "32" // Green
		if msg.Status >= 400 {
			color = "31" // Red
		}

		// Fixed-width printing (no buffering, zero delay)
		// %-12s  = 12 chars wide, left aligned
		// %-6s   = 6 chars wide
		// \033[%sm = ANSI Color start
		// \033[0m  = ANSI Reset
		fmt.Printf("%-12s %-6s %-35s \033[%sm%d OK\033[0m [%s]\n",
			msg.Time,
			msg.Method,
			msg.Path,
			color,
			msg.Status,
			msg.Latency,
		)
	}
}

func saveToHistory(log CombinedLog) {
	historyMutex.Lock()
	defer historyMutex.Unlock()

	// Append to slice
	history = append(history, log)

	// Keep only the latest logs (FIFO)
	if len(history) > maxHistory {
		history = history[1:]
	}
}

// Note: In real code, use context.WithValue(r.Context(), "startTime", time.Now())
