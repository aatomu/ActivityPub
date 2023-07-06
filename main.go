package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
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
	// personTemplate    []byte
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
	// b, err = os.ReadFile("./assets/person.json")
	// if err != nil {
	// 	panic(err)
	// }
	// personTemplate = bytes.ReplaceAll(b, []byte("${Domain}"), []byte(domain))

}

func main() {
	fmt.Printf("Listen:\"%s\" Domain:\"%s\"\n", listenPort, domain)
	// 移動
	_, file, _, _ := runtime.Caller(0)
	goDir := filepath.Dir(file) + "/"
	os.Chdir(goDir)

	// アクセス先
	http.HandleFunc("/.well-known/host-meta", ReturnHostMeta)  // to Webfinger URI
	http.HandleFunc("/.well-known/webfinger", ReturnWebfinger) // return Actor Status URL
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

	// 通常リクエストか
	noClient := strings.Contains(r.Header.Get("Accept"), "application/activity+json")

	switch len(router) {
	case 1: // Top/User Profile URL
		userID := router[0]
		if userID == "" { // https://${Domain}/
			requestLog(r, "ReturnTop()")
			ReturnTop(w, r)
			return
		}
		requestLog(r, "ReturnUserData()") // https://${Domain}/${User}?
		// 存在するユーザか
		if _, err := os.Stat(filepath.Join("./users", userID)); err != nil {
			w.WriteHeader(404)
			return
		}

		switch { // Query Switch
		case r.URL.Query().Has("note"): // Note
			note, err := getNote(userID, r.URL.Query().Get("note"))
			if err != nil {
				w.WriteHeader(500)
				return
			}
			w.Header().Set("Content-Type", "application/activity+json")
			w.Write(note)
			return

		case r.URL.Query().Has("attachment"): // Attachment
			attachment, err := getAttachment(userID, r.URL.Query().Get("attachment"))
			if err != nil {
				w.WriteHeader(500)
				return
			}
			w.Header().Set("Content-Type", http.DetectContentType(attachment))
			w.Write(attachment)
			return
		}

		if noClient { // Get User Person
			person, err := getPerson(userID)
			if err != nil {
				fmt.Println(err)
				w.WriteHeader(500)
				return
			}
			w.Header().Set("Content-Type", "application/activity+json")
			w.Write(person)
			return
		}

		ReturnUserProfile(w, r, userID)
		return

	case 2: // https://${Domain}/${User}/${Event}
		if strings.Contains("abtomu adtomu aetomu actomu", router[0]) || router[0] == "aatomu" {
			return
		}
		requestLog(r, "CatchEvent()")

		// 存在するユーザか
		userID := router[0]
		if _, err := os.Stat(filepath.Join("./users", userID)); err != nil {
			w.WriteHeader(404)
			return
		}

		switch router[1] {
		case "followers":
			if noClient {
				followers, err := getFollowers(userID)
				if err != nil {
					w.WriteHeader(500)
					return
				}
				w.Header().Set("Content-Type", "application/activity+json")
				w.Write(followers)
				return
			}
			w.WriteHeader(501)
			return

		case "following":
			if noClient {
				follows, err := getFollows(userID)
				if err != nil {
					w.WriteHeader(500)
					return
				}
				w.Header().Set("Content-Type", "application/activity+json")
				w.Write(follows)
				return
			}
			w.WriteHeader(501)
			return

		case "icon":
			icon, err := getIcon(userID)
			if err != nil {
				w.WriteHeader(404)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.Write(icon)
			return

		case "inbox":
			// POST以外は対応しない
			if r.Method != "POST" {
				w.WriteHeader(400)
				return
			}

			// Body読み取り
			activity, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(400)
			}
			// Parse Json
			var as ActivityStream
			err = json.Unmarshal(activity, &as)
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
			log.Printf("InboxRequest: Type:\"%s\" Actor:\"%s\" Object:\"%s\"", as.Type, as.Actor, as.Object)
			// 処理
			// Typeに合わせて処理
			switch as.Type {
			case "Follow":
				// 読み込み
				follower, err := getFollowersObject(userID)
				if err != nil {
					log.Println(err)
					w.WriteHeader(500)
					return
				}
				// 加工
				follower.OrderedItems = append(follower.OrderedItems, as.Actor)
				follower.TotalItems = len(follower.OrderedItems)
				// 保存
				err = saveFollowers(userID, follower)
				if err != nil {
					log.Println(err)
					w.WriteHeader(500)
					return
				}

				w.WriteHeader(202)
				// 成功したのを通知
				res, err := Accept(userID, as.Actor, activity)
				if err != nil {
					log.Println(err)
					return
				}
				if res.StatusCode >= 400 && res.StatusCode < 600 {
					log.Println("Failed Accept")
				}
				return

			case "Undo":
				switch as.objectActivity.Type {
				case "Follow":
					// 読み込み
					follower, err := getFollowersObject(userID)
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
					err = saveFollowers(userID, follower)
					if err != nil {
						log.Println(err)
						w.WriteHeader(500)
						return
					}

					w.WriteHeader(200)
					return
				}
			}

		case "outbox":
			outbox, err := getOutbox(userID)
			if err != nil {
				w.WriteHeader(404)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.Write(outbox)
			return
		}

	default: // UnknownURL
		if router[0] == "assets" {
			ReturnAsset(w, r, router[1:])
		}
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
	signatureKeyId := fmt.Sprintf("https://%s/%s#publickey", domain, userID)
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
	accept := r.Header.Get("Accept")
	if len(strings.Split(accept, "")) > 40 {
		accept = strings.Join(strings.Split(accept, "")[:40], "") + "..."
	}
	log.Printf("Access:\"%s\" Catch:\"%s\" Method:\"%s\" URL:\"%s\" Accept:\"%s\" Content-Type:\"%s\"", r.RemoteAddr, catch, r.Method, requestURL, accept, r.Header.Get("Content-Type"))
}
