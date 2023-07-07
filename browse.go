package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"
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
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	passwordBytes, err := os.ReadFile(filepath.Join("./users", userID, "password.sha256"))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if string(passwordBytes) != fmt.Sprintf("%x", sha256.Sum256([]byte(password))) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseMultipartForm(32 << 20)

		var noteText, noteReply string

		if len(r.MultipartForm.Value["note"]) > 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(r.MultipartForm.Value["note"]) == 1 {
			noteText = r.MultipartForm.Value["note"][0]
		}

		if len(r.MultipartForm.Value["reply"]) > 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(r.MultipartForm.Value["reply"]) == 1 {
			noteReply = r.MultipartForm.Value["reply"][0]
		}
		fmt.Println(r.MultipartForm)
		noteSensitive := r.Form.Get("sensitive") == "on"

		noteAttachment := []NoteAttachment{}
		for _, f := range r.MultipartForm.File["attachments"] { // ファイル
			// ファイル読み込み
			file, err := f.Open()
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
			}
			data, err := io.ReadAll(file)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
			}
			// ファイル名
			name := getTimeStamp() + filepath.Ext(f.Filename)
			os.WriteFile(filepath.Join("./users", userID, "attachment", name), data, 0666)
			noteAttachment = append(noteAttachment, NoteAttachment{
				Type:      "Document",
				MediaType: mime.TypeByExtension(filepath.Ext(f.Filename)),
				URL:       fmt.Sprintf("https://%s/%s?attachment=%s", domain, userID, name),
			})
		}

		if noteText != "" || len(noteAttachment) > 0 {
			_, err := createNote(userID, noteText, noteReply, noteSensitive, noteAttachment)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			fmt.Println("ok!")
		}
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

	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(root)))
	w.Write(f)
}

func getTimeStamp() string { //yyyymmddhhMMddssSSSSSS
	return time.Now().Local().Format("20060102150405.000000")
}
