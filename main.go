package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/atomu21263/atomicgo/utils"
)

var (
	listenPort        = os.Args[1]
	domain            = os.Args[2]
	hostMetaTemplate  []byte
	webfingerTemplate []byte
	personTemplate    []byte
)

func init() {
	//// https://${Domain}/.well-known/host-meta
	b, err := os.ReadFile("./assets/.well-known/host-meta.xml")
	if err != nil {
		panic(err)
	}
	hostMetaTemplate = bytes.Replace(b, []byte("${Domain}"), []byte(domain), 1)
	//// https://${Domain}/.well-known/webfinger?resource={uri}
	b, err = os.ReadFile("./assets/.well-known/webfinger.json")
	if err != nil {
		panic(err)
	}
	webfingerTemplate = bytes.ReplaceAll(b, []byte("${Domain}"), []byte(domain))

	// ActivityPub Person Template
	b, err = os.ReadFile("./assets/person.json")
	if err != nil {
		panic(err)
	}
	personTemplate = bytes.ReplaceAll(b, []byte("${Domain}"), []byte(domain))

}

func main() {
	fmt.Printf("Listen:\"%s\" Domain:\"%s\"\n", listenPort, domain)
	// 移動
	_, file, _, _ := runtime.Caller(0)
	goDir := filepath.Dir(file) + "/"
	os.Chdir(goDir)

	// アクセス先
	http.HandleFunc("/.well-known/host-meta", ReturnHostMeta)  // to Webfinger URI
	http.HandleFunc("/.well-known/webfinger", ReturnWebfinger) // return Actor Status
	http.HandleFunc("/", RequestRouter)
	// Web鯖 起動
	go func() {
		log.Println("Http Server Boot")
		err := http.ListenAndServe(listenPort, nil)
		if err != nil {
			log.Println("Failed Listen:", err)
			return
		}
	}()

	<-utils.BreakSignal()
}

// ページ表示
func RequestRouter(w http.ResponseWriter, r *http.Request) {
	router := strings.Split(strings.Replace(r.URL.Path, "/", "", 1), "/")
	fmt.Println(router, len(router))
	switch len(router) {
	case 1: // Top/User Profile URL
		userID := router[0]
		if userID == "" {
			requestLog(r, "ReturnTop()")
			w.WriteHeader(200)
			return
		}
		requestLog(r, "ReturnUserProfile()")
		// 存在するユーザか
		if _, err := os.Stat(filepath.Join("./users", userID)); err != nil {
			w.WriteHeader(404)
			return
		}

		w.WriteHeader(200)
		return
	case 2: // https://${Domain}/${User}/${Event}
		requestLog(r, "CatchEvent()")
		userID := router[0]
		// 存在するユーザか
		if _, err := os.Stat(filepath.Join("./users", userID)); err != nil {
			w.WriteHeader(404)
			return
		}
		switch router[1] {
		case "person":
			CatchPerson(w, r, userID)
		case "followers":
			CatchFollowers(w, r, userID)
		case "following":
			CatchFollowing(w, r, userID)
		case "icon":
			CatchIcon(w, r, userID)
		case "inbox":
			CatchInbox(w, r, userID)
		case "outbox":
			CatchOutbox(w, r, userID)
		}
	default: // UnknownURL
		w.WriteHeader(400)
		return

	}

	w.WriteHeader(404)
}

func GetActorInbox(actor string) (inboxURL string, erro error) {
	// WebfingerURL
	URL, _ := url.Parse(actor)

	requestURL := fmt.Sprintf("%s://%s/.well-known/webfinger?resource=%s", URL.Scheme, URL.Host, actor)
	// Get Webfinger
	resourceResponse, err := HttpRequest("GET", requestURL, nil, map[string]string{})
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
	personResponse, err := HttpRequest("GET", selfURL, nil, map[string]string{"accept": requestType})
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

func HttpRequest(method, url string, body io.Reader, header map[string]string) (*http.Response, error) {
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Set("user-agent", "curl/7.81.0")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	client := new(http.Client)
	return client.Do(req)
}

func requestLog(r *http.Request, catch string) {
	requestURL, _ := url.PathUnescape(r.URL.RequestURI())
	log.Printf("Access:\"%s\" Catch:\"%s\" Method:\"%s\" URL:\"%s\" Content-Type:\"%s\"", r.RemoteAddr, catch, r.Method, requestURL, r.Header.Get("Content-Type"))
}
