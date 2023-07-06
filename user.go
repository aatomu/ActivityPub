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
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func getPerson(userID string) (person []byte, err error) {
	return os.ReadFile(filepath.Join("./users/", userID, "person.json"))
}

func getFollowers(userID string) (followers []byte, err error) {
	return os.ReadFile(filepath.Join("./users", userID, "follower.json"))
}

func getFollowersObject(userID string) (follower ActivityStreamOrderedCollection, err error) {
	followers, err := getFollowers(userID)
	if err != nil {
		return
	}

	err = json.Unmarshal(followers, &follower)
	if err != nil {
		return
	}
	return
}

func saveFollowers(userID string, follower ActivityStreamOrderedCollection) error {
	followerFile := filepath.Join("./users", userID, "follower.json")

	followerList, _ := json.MarshalIndent(follower, "", "  ")
	return os.WriteFile(followerFile, followerList, 0666)
}

func getFollows(userID string) (follows []byte, err error) {
	return os.ReadFile(filepath.Join("./users", userID, "follows.json"))
}

func getIcon(userID string) (icon []byte, err error) {
	return os.ReadFile(filepath.Join("./users", userID, "icon.png"))
}

func getOutbox(userID string) (icon []byte, err error) {
	return os.ReadFile(filepath.Join("./users", userID, "outbox.json"))
}

func Accept(userID, actor string, object []byte) (res *http.Response, err error) {
	// Get Inbox URL
	inboxURL, err := getActorInbox(userID, actor)
	if err != nil {
		return
	}

	// Create Accept Object
	accept := ActivityStream{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    "Accept",
		Actor:   fmt.Sprintf("https://%s/%s/person", domain, userID),
		Object:  "${Object}",
	}
	acceptBytes, err := json.Marshal(accept)
	if err != nil {
		return
	}
	// Replace DummyData To ActivityObject
	acceptBytes = bytes.Replace(acceptBytes, []byte("\"${Object}\""), object, 1)

	// Http Request 作成
	req, _ := http.NewRequest("POST", inboxURL, bytes.NewReader(acceptBytes))
	req.Header.Set("user-agent", "original/1.1.1")
	req.Header.Set("accept", "application/json")
	requestDate := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("date", requestDate)

	// 秘密鍵 読み込み
	privateKeyBytes, err := os.ReadFile(filepath.Join("./users", userID, "privatekey.pem"))
	if err != nil {
		return
	}
	privateKeyBlock, _ := pem.Decode(privateKeyBytes)

	privateKeyAny, err := x509.ParsePKCS8PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		return nil, err
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
		return
	}
	req.Header.Set("signature", fmt.Sprintf("keyId=\"%s\",algorithm=\"rsa-sha256\",headers=\"%s\",signature=\"%s\"", signatureKeyId, signatureHeaders, signatureData))

	// Sent Actor Inbox
	client := new(http.Client)
	return client.Do(req)
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
