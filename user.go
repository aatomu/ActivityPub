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

	var ok bool
	as.objectStr, ok = as.Object.(string) // as.objectがstringにキャスト可能か
	if !ok {                              // 出来なかったらObjectにキャスト
		j, _ := json.Marshal(as.Object)
		json.Unmarshal(j, &as.objectActivity)
	}

	// Typeに合わせて処理
	switch as.Type {
	case "Follow":
		// 読み込み
		follower, err := GetFollower(userID)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}
		// 加工
		follower.OrderedItems = append(follower.OrderedItems, as.Actor)
		follower.TotalItems = len(follower.OrderedItems)
		// 保存
		err = SaveFollower(userID, follower)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}

		// 成功したのを通知
		err = Accept(userID, as.Actor, request)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}
		return

	case "Undo":
		switch as.objectActivity.Type {
		case "Follow":
			// 読み込み
			follower, err := GetFollower(userID)
			if err != nil {
				log.Println(err)
				w.WriteHeader(500)
				return
			}
			// 加工
			newFollower := []string{}
			for _, v := range follower.OrderedItems {
				if v == as.objectActivity.Actor {
					continue
				}
				newFollower = append(newFollower, v)
			}
			follower.OrderedItems = newFollower
			follower.TotalItems = len(follower.OrderedItems)
			// 保存
			err = SaveFollower(userID, follower)
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

func Accept(userID, actor string, object []byte) error {
	// Get Inbox URL
	inboxURL, err := GetActorInbox(actor)
	if err != nil {
		return err
	}

	// Create Accept Object
	accept := ActivityStream{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    "Accept",
		Actor:   userID,
		Object:  "${Object}",
	}
	acceptBytes, err := json.Marshal(accept)
	if err != nil {
		return err
	}
	// Replace DummyData To ActivityObject
	acceptBytes = bytes.Replace(acceptBytes, []byte("${Object}"), object, 1)

	// 署名

	// Sent Actor Inbox
	res, err := HttpRequest("POST", inboxURL, bytes.NewReader(acceptBytes), map[string]string{})
	if err != nil {
		return err
	}
	fmt.Printf("%+v\n", res)
	return nil
}
