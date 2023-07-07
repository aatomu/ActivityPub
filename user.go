package main

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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

func getOutbox(userID string) (outbox []byte, err error) {
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
		Actor:   fmt.Sprintf("https://%s/%s", domain, userID),
		Object:  "${Object}",
	}
	acceptBytes, err := json.Marshal(accept)
	if err != nil {
		return
	}
	// Replace DummyData To ActivityObject
	acceptBytes = bytes.Replace(acceptBytes, []byte("\"${Object}\""), object, 1)
	return HttpRequest(http.MethodPost, userID, inboxURL, acceptBytes, map[string]string{"accept": "application/activity+json"})
}

func getActorInbox(userID, actor string) (inboxURL string, erro error) {
	// WebfingerURL
	URL, _ := url.Parse(actor)

	requestURL := fmt.Sprintf("%s://%s/.well-known/webfinger?resource=%s", URL.Scheme, URL.Host, actor)
	// Get Webfinger
	resourceResponse, err := HttpRequest("GET", userID, requestURL, []byte{}, map[string]string{})
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
	personResponse, err := HttpRequest("GET", userID, selfURL, []byte{}, map[string]string{"accept": requestType})
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

func createNote(userID, noteText, reply string, sensitive bool, attachments []NoteAttachment) (noteBytes []byte, err error) {
	t := getTimeStamp()
	note := Note{
		Context:      "https://www.w3.org/ns/activitystreams",
		ID:           fmt.Sprintf("https://%s/%s?note=%s", domain, userID, t),
		Type:         "Note",
		InReplyTo:    reply,
		Published:    time.Now().UTC().Format(http.TimeFormat),
		URL:          fmt.Sprintf("https://%s/%s?note=%s", domain, userID, t),
		AttributedTo: fmt.Sprintf("https://%s/%s", domain, userID),
		Content:      fmt.Sprintf("<p>%s<p>", noteText),
		To: []string{
			"https://www.w3.org/ns/activitystreams#Public",
			fmt.Sprintf("https://%s/%s/followers", domain, userID),
		},
		Sensitive:  sensitive,
		Attachment: attachments,
	}

	noteBytes, err = json.MarshalIndent(note, "", "  ")
	if err != nil {
		return
	}

	err = os.WriteFile(filepath.Join("./users", userID, "note", t+".json"), noteBytes, 0666)
	if err != nil {
		return
	}

	return
}
