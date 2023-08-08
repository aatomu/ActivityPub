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
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func getPerson(userID string) (person []byte, err error) {
	return os.ReadFile(filepath.Join("./users/", userID, "person.json"))
}

func getFollowers(userID string) (followers []byte, err error) {
	return os.ReadFile(filepath.Join("./users", userID, "follower.json"))
}

func getFollowersObject(userID string) (follower ActivityStream, err error) {
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

func saveFollowers(userID string, follower ActivityStream) error {
	followerFile := filepath.Join("./users", userID, "follower.json")

	followerList, _ := ToJson(follower)
	return os.WriteFile(followerFile, followerList, 0666)
}

func getFollows(userID string) (follows []byte, err error) {
	return os.ReadFile(filepath.Join("./users", userID, "follows.json"))
}

func getIcon(userID string) (icon []byte, err error) {
	return os.ReadFile(filepath.Join("./users", userID, "icon.png"))
}

func getOutbox(userID string, page string) (outbox []byte, err error) {
	d, err := filepath.Glob(filepath.Join("./users", userID, "note", "*.json"))
	if err != nil {
		return
	}

	if page == "" {
		out := ActivityStream{
			Context:    "https://www.w3.org/ns/activitystreams",
			ID:         fmt.Sprintf("https://%s/%s/outbox", domain, userID),
			Type:       "OrderedCollection",
			TotalItems: len(d),
			First:      fmt.Sprintf("https://%s/%s/outbox?page=%d", domain, userID, 0),
			Last:       fmt.Sprintf("https://%s/%s/outbox?page=%d", domain, userID, (len(d)+20)/20),
		}
		return ToJson(out)
	}

	pageNum, err := strconv.Atoi(page)
	if err != nil {
		return
	}

	var list []interface{}
	var note Note
	for i := pageNum * 20; i < len(d) && i < pageNum*20+20; i++ {
		noteBytes, err := os.ReadFile(d[i])
		if err != nil {
			return []byte{}, err
		}
		json.Unmarshal(noteBytes, &note)
		list = append(list, ActivityStream{
			Context: "https://www.w3.org/ns/activitystreams",
			Type:    "Create",
			Object:  note,
		})
	}

	var next, prev string
	if len(d) > pageNum+1*20 {
		next = fmt.Sprintf("https://%s/%s/outbox?page=%d", domain, userID, pageNum+1)
	}
	if len(d)-pageNum-1*20 > 0 {
		prev = fmt.Sprintf("https://%s/%s/outbox?page=%d", domain, userID, pageNum-1)
	}
	collection := ActivityStream{
		Context:      "https://www.w3.org/ns/activitystreams",
		ID:           fmt.Sprintf("https://%s/%s/outbox?page=%s", domain, userID, page),
		Type:         "OrderedCollectionPage",
		Next:         next,
		Prev:         prev,
		PartOf:       fmt.Sprintf("https://%s/%s/outbox", domain, userID),
		OrderedItems: list,
	}
	return ToJson(collection)
}

func inboxEventFollow(w http.ResponseWriter, userID, actor string, activity []byte) {
	// 読み込み
	follower, err := getFollowersObject(userID)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	// 加工
	newFollowers := append(follower.OrderedItems, actor)
	var items []interface{}
	m := make(map[string]bool)
	for _, actor := range newFollowers { // 重複回避
		if !m[actor.(string)] {
			m[actor.(string)] = true
			items = append(items, actor)
		}
	}
	follower.OrderedItems = items
	follower.TotalItems = len(follower.OrderedItems)
	// 保存
	err = saveFollowers(userID, follower)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	// 成功したのを通知
	res, err := Accept(userID, actor, activity)
	if err != nil {
		log.Println(err)
		return
	}
	if res.StatusCode >= 400 && res.StatusCode < 600 {
		log.Println("Failed Accept")
	}
}

func inboxEventAccept(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Header.Get("Signature") == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	fmt.Println(r.Header.Values("Signature"))
}

func inboxEventUndo(w http.ResponseWriter, userID string, undoActivity ActivityStream) {
	switch undoActivity.Type {
	case "Follow":
		// 読み込み
		follower, err := getFollowersObject(userID)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusNotImplemented)
			return
		}
		// 加工
		var newFollower []interface{}
		for _, v := range follower.OrderedItems {
			if v == undoActivity.Actor {
				continue
			}
			newFollower = append(newFollower, v)
		}
		follower.OrderedItems = newFollower
		follower.TotalItems = len(follower.OrderedItems)
		// 保存
		err = saveFollowers(userID, follower)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		return
	default:
		w.WriteHeader(http.StatusNotImplemented)
	}
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

	noteBytes, err = ToJson(note)
	if err != nil {
		return
	}

	err = os.WriteFile(filepath.Join("./users", userID, "note", t+".json"), noteBytes, 0666)
	if err != nil {
		return
	}

	return
}
