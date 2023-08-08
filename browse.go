package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func ReturnTop(w http.ResponseWriter, r *http.Request, title string) {
	userID, isAutorized := authorize(r)
	if r.Method == http.MethodPost && isAutorized {
		saveNote(w, r, userID)
		return
	}

	f, err := os.ReadFile(filepath.Join("./assets", "index.html"))
	if err != nil {
		w.WriteHeader(404)
		return
	}

	f = bytes.ReplaceAll(f, []byte("${Domain}"), []byte(domain))
	f = bytes.ReplaceAll(f, []byte("${Title}"), []byte(title))
	f = bytes.ReplaceAll(f, []byte("${Owner}"), []byte(owener))
	if isAutorized {
		ops, err := os.ReadFile(filepath.Join("./assets", "authorized.html"))
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		f = bytes.ReplaceAll(f, []byte("${Authorized}"), ops)
	} else {
		f = bytes.ReplaceAll(f, []byte("${Authorized}"), []byte{})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(f)
}

func authorize(r *http.Request) (userID string, authorized bool) {
	nameCookie, err := r.Cookie("name")
	if err == nil {
		userID = nameCookie.Value
	}

	passwdCookie, err := r.Cookie("passwd")
	if err == nil {
		_, err := os.Stat(filepath.Join("./users", userID))
		if err == nil { // is EnableUser

			passwordBytes, err := os.ReadFile(filepath.Join("./users", userID, "password.sha256"))
			if err == nil {
				authorized = string(passwordBytes) == fmt.Sprintf("%x", sha256.Sum256([]byte(passwdCookie.Value)))
			}
		}
	}
	return
}

func saveNote(w http.ResponseWriter, r *http.Request, userID string) {
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
		noteBytes, err := createNote(userID, noteText, noteReply, noteSensitive, noteAttachment)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		create := ActivityStream{
			Context: "https://www.w3.org/ns/activitystreams",
			Type:    "Create",
			Object:  "${Object}",
		}
		createBytes, err := json.Marshal(create)
		if err != nil {
			log.Println(err)
			return
		}
		createBytes = bytes.Replace(createBytes, []byte("\"${Object}\""), noteBytes, 1)

		followers, err := getFollowersObject(userID)
		if err != nil {
			log.Println(err)
			return
		}
		for _, actorID := range followers.OrderedItems {
			url, err := getActorInbox(userID, actorID.(string))
			if err != nil {
				continue
			}

			HttpRequest("POST", userID, url, createBytes, map[string]string{"Content-Type": "application/activity+json"})
		}

	}
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
