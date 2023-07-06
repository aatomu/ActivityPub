package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

func ReturnTop(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Has("view") {
		f, err := os.ReadFile(filepath.Join("./assets", "view_only.html"))
		if err != nil {
			w.WriteHeader(404)
			return
		}
		f = bytes.ReplaceAll(f, []byte("${Domain}"), []byte(domain))
		f = bytes.ReplaceAll(f, []byte("${Owner}"), []byte(owener))

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(f)
		return
	}

	// request Auth
	w.Header().Set("WWW-Authenticate", `Basic realm="Check Login User"`)
	userID, password, authOK := r.BasicAuth()

	if !authOK { // Failed Auth
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if _, err := os.Stat(filepath.Join("./users", userID)); err != nil { // is EnableUser
		w.WriteHeader(http.StatusForbidden)
		return
	}

	passwordBytes, err := os.ReadFile(filepath.Join("./users", userID, "password.sha256"))
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if string(passwordBytes) != fmt.Sprintf("%x", sha256.Sum256([]byte(password))) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

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
