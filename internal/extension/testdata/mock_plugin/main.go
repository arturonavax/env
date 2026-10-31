package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type rpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func main() {
	exeName := filepath.Base(os.Args[0])
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(exeName, "corrupted") {
			fmt.Println("GARBAGE NOT JSON DATA")
			continue
		}
		var req rpcReq
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}
		if strings.Contains(exeName, "envprobe") {
			cwd, _ := os.Getwd()
			name := fmt.Sprintf("secret=%q;cwd=%s", os.Getenv("ANX_TEST_SECRET"), cwd)
			out, _ := json.Marshal(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]any{
					"name": name, "version": "1.0", "protocol_version": "1.0", "permissions": []string{},
				},
			})
			fmt.Println(string(out))
			continue
		}
		if strings.Contains(exeName, "flood") {
			fmt.Println(strings.Repeat("A", 4*1024*1024)) // single oversized message
			continue
		}
		if strings.Contains(exeName, "incompatible") {
			fmt.Printf("{\"jsonrpc\":\"2.0\",\"id\":%d,\"result\":{\"name\":\"bad\",\"version\":\"2.0\",\"protocol_version\":\"99.0\",\"permissions\":[]}}\n", req.ID)
			continue
		}
		if strings.Contains(exeName, "unpermitted") {
			fmt.Printf("{\"jsonrpc\":\"2.0\",\"id\":%d,\"result\":{\"name\":\"evil\",\"version\":\"1.0\",\"protocol_version\":\"1.0\",\"permissions\":[\"root_access\"]}}\n", req.ID)
			continue
		}
		if strings.Contains(exeName, "hanging") {
			if req.Method == "initialize" {
				fmt.Printf("{\"jsonrpc\":\"2.0\",\"id\":%d,\"result\":{\"name\":\"hang\",\"version\":\"1.0\",\"protocol_version\":\"1.0\",\"permissions\":[]}}\n", req.ID)
			} else {
				time.Sleep(30 * time.Second)
			}
			continue
		}

		if req.Method == "initialize" {
			fmt.Printf("{\"jsonrpc\":\"2.0\",\"id\":%d,\"result\":{\"name\":\"test-plugin\",\"version\":\"0.1.0\",\"protocol_version\":\"1.0\",\"permissions\":[\"read\"],\"capabilities\":[\"echo\"]}}\n", req.ID)
		} else if req.Method == "execute" {
			fmt.Printf("{\"jsonrpc\":\"2.0\",\"id\":%d,\"result\":{\"exit_code\":0,\"output\":\"hello from plugin\"}}\n", req.ID)
		} else {
			fmt.Printf("{\"jsonrpc\":\"2.0\",\"id\":%d,\"error\":{\"code\":-32601,\"message\":\"Method not found\"}}\n", req.ID)
		}
	}
}
