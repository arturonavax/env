package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	cmdName := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	args := os.Args[1:]
	logFile := os.Getenv("ANX_MOCK_LOG_FILE")

	switch cmdName {
	case "go":
		if len(args) >= 3 && args[0] == "version" && args[1] == "-m" {
			target := filepath.Base(args[2])
			switch {
			case strings.Contains(target, "gopls"):
				fmt.Printf("%s: go1.27.3\n\tpath\tgolang.org/x/tools/gopls\n\tmod\tgolang.org/x/tools/gopls\tv0.18.0\n", args[2])
			case strings.Contains(target, "gofumpt"):
				fmt.Printf("%s: go1.27.3\n\tpath\tmvdan.cc/gofumpt\n\tmod\tmvdan.cc/gofumpt\tv0.7.0\n", args[2])
			case strings.Contains(target, "localdev"):
				fmt.Printf("%s: go1.27.3\n\tpath\tcommand-line-arguments\n", args[2])
			default:
				fmt.Printf("%s: go1.27.3\n\tpath\tcommand-line-arguments\n", args[2])
			}
			return
		}
		if logFile != "" {
			f, _ := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if f != nil {
				fmt.Fprintf(f, "go %s\n", strings.Join(args, " "))
				_ = f.Close()
			}
		}

	case "cargo":
		if len(args) >= 2 && args[0] == "install" && args[1] == "--list" {
			fmt.Print("difftastic v0.61.0:\n    difft\ngit-delta v0.18.2:\n    delta\nlocal-tool v0.1.0 (/tmp/local-tool):\n    local-tool\n")
			return
		}
		if logFile != "" {
			f, _ := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if f != nil {
				fmt.Fprintf(f, "cargo %s\n", strings.Join(args, " "))
				_ = f.Close()
			}
		}

	case "pnpm", "npm", "pipx", "nvim", "zsh", "rustup":
		if logFile != "" {
			f, _ := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if f != nil {
				fmt.Fprintf(f, "%s %s\n", cmdName, strings.Join(args, " "))
				_ = f.Close()
			}
		}

	case "lynis":
		var reportFile string
		for i := 0; i < len(args); i++ {
			if args[i] == "--report-file" && i+1 < len(args) {
				reportFile = args[i+1]
				i++
			}
		}
		if reportFile != "" {
			content := "hardening_index=74\nwarning[]=FIRE-4512|Firewall module inactive|\nsuggestion[]=KRNL-6000|Restrict kernel pointer exposure via sysctl|\n"
			_ = os.WriteFile(reportFile, []byte(content), 0o644)
		}
		fmt.Println("[WARNING] Check firewall rules")

	case "bash":
		if len(args) >= 2 && args[0] == "-n" {
			return
		}
		if len(args) >= 1 {
			scriptPath := args[len(args)-1]
			data, err := os.ReadFile(scriptPath)
			if err == nil {
				lines := strings.Split(string(data), "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "echo ") && strings.Contains(line, " > ") {
						parts := strings.SplitN(line, " > ", 2)
						val := strings.Trim(strings.TrimPrefix(parts[0], "echo "), "'\"")
						dest := strings.Trim(parts[1], "'\"")
						_ = os.WriteFile(dest, []byte(val+"\n"), 0o644)
					}
				}
			}
			return
		}
	}
}
