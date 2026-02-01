# ProxyEye 👁️

**ProxyEye** is a lightweight, high-performance HTTP reverse proxy and request inspector built in Go. It bridges local traffic to the public internet via Cloudflare Tunnels while providing a powerful web dashboard to inspect every detail of your API traffic.

```
  _____                      ______             
 |  __ \                    |  ____|            
 | |__) | __ _____  ___   _ | |__  _   _  ___   
 |  ___/ '__/ _ \ \/ / | | ||  __|| | | |/ _ \  
 | |   | | | (_) >  <| |_| || |___| |_| |  __/  
 |_|   |_|  \___/_/\_\\__, ||______\__, |\___|  
                       __/ |        __/ |       
                      |___/        |___/        

```

---

## ✨ Features

* **Real-time Inspection:** Live-streaming logs via WebSockets to a modern web dashboard.

---

## 🚀 Installation

### 1. Prerequisites

* **Go 1.20+**

### 2. Build from Source

```bash
# Clone the repository
git clone [https://github.com/yannice92/proxyeye.git](https://github.com/yannice92/proxyeye.git)

# Move into the directory
cd proxyeye

# Build the portable binary
go build -o proxyeye

```

---

## 🛠️ Usage

Point ProxyEye to your local application port (e.g., if your app is running on port 3000):

```bash
./proxyeye 3000

```

### Options & Flags

| Flag | Description | Default |
| --- | --- | --- |
| `-p` | The target port of your local application. | `3000` |
| `--domain` | Custom local domain mapping. | `localhost` |
| `--ui-port` | Port for the Web Inspector UI. | `4040` |

---

## 🖥️ The Dashboard

### Web Inspector

Navigate to `http://localhost:4040/inspect` to access:

* **Request/Response Tabs:** Organized metadata for clear auditing.
* **JSON Viewer:** Formatted and colored JSON payloads for better readability.
* **Latency Tracking:** Precisely measured request-to-response duration in milliseconds.

### CLI View

The terminal provides a live-scrolling feed of incoming requests with immediate feedback:

```text
20:55:01  GET  /api/data      200 OK [1.24ms]
20:55:10  POST /api/login     401 OK [15.50ms]

```

---

## 🛡️ License

Distributed under the MIT License.

```

---

Since you are wrapping up your repository, would you like me to help you generate the **LICENSE** file (MIT) text to match?

This [video guide on exporting Gemini chats](https://www.youtube.com/watch?v=JAfR9xrRgwA) shows how you can use Canvas mode to export your conversations directly into formatted documents or PDFs for your records.


http://googleusercontent.com/youtube_content/21

```