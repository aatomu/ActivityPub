package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func CatchPerson(w http.ResponseWriter, r *http.Request, userID string) { // /${User}/person
	pubKey, err := os.ReadFile(filepath.Join("./users/", userID, "publickey.pem"))
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
	}
	pubKey = bytes.ReplaceAll(pubKey, []byte("\n"), []byte("\\n"))

	person := bytes.ReplaceAll(personTemplate, []byte("${User}"), []byte(userID))
	person = bytes.Replace(person, []byte("${PublicKey}"), pubKey, 1)

	w.Header().Set("Content-Type", "application/activity+json")
	w.Write(person)
}

func CatchFollowers(w http.ResponseWriter, r *http.Request, userID string) { // /${User}/followers
}

func CatchFollowing(w http.ResponseWriter, r *http.Request, userID string) { // /${User}/following
}

func CatchIcon(w http.ResponseWriter, r *http.Request, userID string) { // /${User}/icon
	f, err := os.ReadFile(filepath.Join("./users", userID, "icon.png"))
	if err != nil {
		w.WriteHeader(404)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(f)
}

func CatchInbox(w http.ResponseWriter, r *http.Request, userID string) { // /${User}/inbox
	// POST以外は対応しない
	if r.Method != "POST" {
		w.WriteHeader(400)
		return
	}
	// Body読み取り
	request, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(400)
	}
	log.Println("InboxRequest:", string(request))
	// Jsonにパーズ
	var as ActivityStream
	err = json.Unmarshal(request, &as)
	if err != nil {
		log.Println(err)
		w.WriteHeader(400)
		return
	}
	switch v := as.Object.(type) {
	case string:
		as.objectStr = v
	default:
		j, _ := json.Marshal(v)
		json.Unmarshal(j, &as.objectActivity)
	}

	// Typeに合わせて処理
	switch as.Type {
	case "Follow":
		// フォロワー 一覧 入手
		followerFile := filepath.Join("./users", userID, "follower.json")
		followerList, err := os.ReadFile(followerFile)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}
		var followers ActivityStreamOrderedCollection
		err = json.Unmarshal(followerList, &followers)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}
		followers.OrderedItems = append(followers.OrderedItems, as.Actor)
		followers.TotalItems = len(followers.OrderedItems)
		followerList, _ = json.MarshalIndent(followers, "", "  ")
		err = os.WriteFile(followerFile, followerList, 0666)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}

		// 成功したのを通知
		inboxURL, err := GetUserInbox(as.Actor)
		if err != nil {
			log.Println(err)
			return
		}
		accept := ActivityStream{
			Context: "https://www.w3.org/ns/activitystreams",
			Type:    "Accept",
			Actor:   userID,
			Object:  "${Object}",
		}
		acceptBytes, err := json.Marshal(accept)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
		}
		acceptBytes = bytes.Replace(acceptBytes, []byte("${Object}"), request, 1)
		fmt.Println(string(acceptBytes))
		res, err := HttpRequest("POST", inboxURL, bytes.NewReader(acceptBytes), map[string]string{})
		if err != nil {
			log.Println(err)
		}
		fmt.Printf("%+v\n", res)
		return

	case "Undo":
		switch as.objectActivity.Type {
		case "Follow":
			// フォロワーを保存
			followerFile := filepath.Join("./users", userID, "follower.json")
			followerList, err := os.ReadFile(followerFile)
			if err != nil {
				log.Println(err)
				w.WriteHeader(500)
				return
			}
			var followers ActivityStreamOrderedCollection
			err = json.Unmarshal(followerList, &followers)
			if err != nil {
				log.Println(err)
				w.WriteHeader(500)
				return
			}
			newFollowers := []string{}
			for _, v := range followers.OrderedItems {
				if v == as.objectActivity.Actor {
					continue
				}
				newFollowers = append(newFollowers, v)
			}
			followers.OrderedItems = newFollowers
			followers.TotalItems = len(followers.OrderedItems)
			followerList, _ = json.MarshalIndent(followers, "", "  ")
			err = os.WriteFile(followerFile, followerList, 0666)
			if err != nil {
				log.Println(err)
				w.WriteHeader(500)
				return
			}
			w.WriteHeader(200)
			return
		}
	}
}

func CatchOutbox(w http.ResponseWriter, r *http.Request, userID string) { // /${User}/outbox
}
