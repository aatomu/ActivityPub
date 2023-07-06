package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/atomu21263/atomicgo/utils"
)

var (
	listenPort        = os.Args[1]
	domain            = os.Args[2]
	owener            = os.Args[3]
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
	// 条件分岐用
	router := strings.Split(strings.Replace(r.URL.Path, "/", "", 1), "/")

	switch len(router) {
	case 1: // Top/User Profile URL
		userID := router[0]
		if userID == "" { // https://${Domain}/
			requestLog(r, "ReturnTop()")
			ReturnTop(w, r)
			w.WriteHeader(200)
			return
		}
		requestLog(r, "ReturnUserProfile()") // https://${Domain}/${User}
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
		if router[0] == "assets" {
			ReturnAsset(w, r, router[1:])
		}
		w.WriteHeader(400)
		return

	}

	w.WriteHeader(404)
}

func HttpGetRequest(method, userID, url string, body []byte, header map[string]string) (*http.Response, error) {
	// Http Request 作成
	req, _ := http.NewRequest(method, url, bytes.NewReader(body))
	req.Header.Set("user-agent", "original/1.1.1")
	requestDate := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("date", requestDate)
	for k, v := range header {
		req.Header.Set(k, v)
	}

	// 秘密鍵 読み込み
	privateKeyBytes, err := os.ReadFile(filepath.Join("./users", userID, "privatekey.pem"))
	if err != nil {
		return nil, err
	}
	privateKeyBlock, _ := pem.Decode(privateKeyBytes)

	privateKeyAny, err := x509.ParsePKCS8PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		return nil, err
	}
	privateKey := privateKeyAny.(*rsa.PrivateKey)

	// digest header 生成
	digest := createDigest(body)
	req.Header.Set("digest", digest)

	// signature header 作成
	signatureKeyId := fmt.Sprintf("https://%s/%s/person#publickey", domain, userID)
	signatureHeaders := "(request-target) host date digest"

	degestHeader := fmt.Sprintf("(request-target): %s %s\nhost: %s\ndate: %s", strings.ToLower(method), req.URL.Path, req.Host, requestDate)
	if method == "POST" {
		degestHeader += fmt.Sprintf("\ndigest: %s", digest)
	}
	signatureData, err := createSignature([]byte(degestHeader), privateKey)
	if err != nil {
		return nil, err
	}
	req.Header.Set("signature", fmt.Sprintf("keyId=\"%s\",algorithm=\"rsa-sha256\",headers=\"%s\",signature=\"%s\"", signatureKeyId, signatureHeaders, signatureData))

	// Sent Actor Inbox
	client := new(http.Client)
	_, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func requestLog(r *http.Request, catch string) {
	requestURL, _ := url.PathUnescape(r.URL.RequestURI())
	log.Printf("Access:\"%s\" Catch:\"%s\" Method:\"%s\" URL:\"%s\" Content-Type:\"%s\"", r.RemoteAddr, catch, r.Method, requestURL, r.Header.Get("Content-Type"))
}
