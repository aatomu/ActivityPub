package main

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
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
	inboxURL, err := getActorInbox(userID, actor)
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

	// Http Request 作成
	req, _ := http.NewRequest("POST", inboxURL, bytes.NewReader(acceptBytes))
	req.Header.Set("user-agent", "original/1.1.1")
	req.Header.Set("accept", "application/json")
	requestDate := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("date", requestDate)

	// 秘密鍵 読み込み
	privateKeyBytes, err := os.ReadFile(filepath.Join("./users", userID, "privatekey.pem"))
	if err != nil {
		return err
	}
	privateKeyBlock, _ := pem.Decode(privateKeyBytes)

	privateKeyAny, err := x509.ParsePKCS8PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		return err
	}
	privateKey := privateKeyAny.(*rsa.PrivateKey)

	// digest header 生成
	digest := createDigest(acceptBytes)
	req.Header.Set("digest", digest)

	// signature header 作成
	signatureKeyId := fmt.Sprintf("https://%s/%s/person#publickey", domain, userID)
	signatureHeaders := "(request-target) host date digest"

	degestHeader := fmt.Sprintf("(request-target): post %s\nhost: %s\ndate: %s\ndigest: %s", req.URL.Path, req.Host, requestDate, digest)
	signatureData, err := createSignature([]byte(degestHeader), privateKey)
	if err != nil {
		return err
	}
	req.Header.Set("signature", fmt.Sprintf("keyId=\"%s\",algorithm=\"rsa-sha256\",headers=\"%s\",signature=\"%s\"", signatureKeyId, signatureHeaders, signatureData))

	// Sent Actor Inbox
	client := new(http.Client)
	_, err = client.Do(req)
	if err != nil {
		return err
	}
	return nil
}

func getActorInbox(userID, actor string) (inboxURL string, erro error) {
	// WebfingerURL
	URL, _ := url.Parse(actor)

	requestURL := fmt.Sprintf("%s://%s/.well-known/webfinger?resource=%s", URL.Scheme, URL.Host, actor)
	// Get Webfinger
	resourceResponse, err := HttpGetRequest("GET", userID, requestURL, []byte{}, map[string]string{})
	if err != nil {
		return "", err
	}
	resourceBytes, err := io.ReadAll(resourceResponse.Body)
	if err != nil {
		return "", err
	}
	var resource Resource
	json.Unmarshal(resourceBytes, &resource)
	if err != nil {
		return "", err
	}
	var selfURL, requestType string
	for _, v := range resource.Links {
		if v.Rel == "self" {
			selfURL = v.Href
			requestType = v.Type
		}
	}
	// Get Person
	personResponse, err := HttpGetRequest("GET", userID, selfURL, []byte{}, map[string]string{"accept": requestType})
	if err != nil {
		return "", err
	}
	personBytes, err := io.ReadAll(personResponse.Body)
	if err != nil {
		return "", err
	}
	var person Person
	json.Unmarshal(personBytes, &person)
	if err != nil {
		return "", err
	}
	return person.Inbox, nil
}

func createDigest(body []byte) (digest string) {
	hash := sha256.Sum256(body)
	digest = fmt.Sprintf("sha-256=%s", base64.StdEncoding.EncodeToString(hash[:]))
	return
}

func createSignature(data []byte, privateKey *rsa.PrivateKey) (signature string, err error) {
	hash := sha256.Sum256(data)

	// digest生成(sha256,SRA-PKCS1-v1_5)
	digestBytes, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return
	}

	return base64.StdEncoding.EncodeToString(digestBytes), nil
}
