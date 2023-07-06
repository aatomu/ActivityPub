package main

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
)

func ReturnTop(w http.ResponseWriter, r *http.Request) {
	f, err := os.ReadFile(filepath.Join("./assets", "index.html"))
	if err != nil {
		w.WriteHeader(404)
		return
	}
	f = bytes.ReplaceAll(f, []byte("${Domain}"), []byte(domain))
	f = bytes.ReplaceAll(f, []byte("${Owner}"), []byte(owener))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(f)
}

func ReturnUserProfile(w http.ResponseWriter, r *http.Request, userID string) {

}

func ReturnAsset(w http.ResponseWriter, r *http.Request, path []string) {
	requestLog(r, "ReturnAsset()")
	root := filepath.Join(path...)
	f, err := os.ReadFile(filepath.Join("./assets", root))
	if err != nil {
		w.WriteHeader(404)
		return
	}

	switch filepath.Ext(root) {
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	default:
		w.Header().Set("Content-Type", http.DetectContentType(f))
	}
	w.Write(f)
}
