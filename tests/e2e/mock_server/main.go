package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	repoDir := os.Getenv("REPO_DIR")
	if repoDir == "" {
		repoDir = "/repo"
	}
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "/tmp/mock-releases"
	}

	_ = os.MkdirAll(staticDir, 0o755)

	tarballPath := filepath.Join(staticDir, "anx_linux_amd64.tar.gz")

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK\n"))
	})

	// Serve install.sh dynamically from repository
	mux.HandleFunc("/install.sh", func(w http.ResponseWriter, r *http.Request) {
		installPath := filepath.Join(repoDir, "install.sh")
		if data, err := os.ReadFile(installPath); err == nil {
			w.Header().Set("Content-Type", "text/x-shellscript")
			_, _ = w.Write(data)
			return
		}
		http.Error(w, "install.sh not found", http.StatusNotFound)
	})

	// Serve GitHub release tarballs
	mux.HandleFunc("/releases/latest/download/", func(w http.ResponseWriter, r *http.Request) {
		filename := strings.TrimPrefix(r.URL.Path, "/releases/latest/download/")
		filePath := filepath.Join(staticDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			filePath = tarballPath
		}
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// Generate on demand if missing
			if err := createReleaseTarball(repoDir, filePath); err != nil {
				log.Printf("[ERR] Failed to create release tarball on demand: %v", err)
				http.Error(w, fmt.Sprintf("Failed to generate release: %v", err), http.StatusInternalServerError)
				return
			}
		}
		if _, err := os.Stat(filePath); err == nil {
			w.Header().Set("Content-Type", "application/gzip")
			http.ServeFile(w, r, filePath)
			return
		}
		http.Error(w, "Release asset not found", http.StatusNotFound)
	})

	log.Printf("[INFO] Mock Release & Installer Server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server exited: %v", err)
	}
}

func createReleaseTarball(repoDir, destTarball string) error {
	tmpDir, err := os.MkdirTemp("", "mock-build-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	binPath := filepath.Join(tmpDir, "anx")

	// If a pre-built anx binary already exists in repo/bin/anx, reuse it
	repoBin := filepath.Join(repoDir, "bin", "anx")
	if info, sErr := os.Stat(repoBin); sErr == nil && !info.IsDir() {
		data, rErr := os.ReadFile(repoBin)
		if rErr != nil {
			return rErr
		}
		if err := os.WriteFile(binPath, data, 0o755); err != nil {
			return err
		}
	} else {
		// Compile it using go build
		cmd := exec.Command("go", "build", "-o", binPath, "./cmd/anx")
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
		if out, bErr := cmd.CombinedOutput(); bErr != nil {
			return fmt.Errorf("building anx: %v (%s)", bErr, string(out))
		}
	}

	// Create tar.gz containing anx
	outFile, err := os.Create(destTarball)
	if err != nil {
		return err
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	binData, err := os.ReadFile(binPath)
	if err != nil {
		return err
	}

	hdr := &tar.Header{
		Name: "anx",
		Mode: 0o755,
		Size: int64(len(binData)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write(binData); err != nil {
		return err
	}

	return nil
}
