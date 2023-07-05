package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

func ReturnHostMeta(w http.ResponseWriter, r *http.Request) { // /.well-known/host-meta
	requestLog(r, "ReturnHostMeta()")

	w.Header().Set("Content-Type", "application/xml")
	w.Write(hostMetaTemplate)
}

func ReturnWebfinger(w http.ResponseWriter, r *http.Request) { // /.well-known/host-meta
	requestLog(r, "ReturnWebfinger()")

	resource := r.URL.Query().Get("resource")
	var userID string
	switch {
	case strings.HasPrefix(resource, "acct:"): // acct:${User}@{Domain}
		userID = strings.Split(resource, "@"+domain)[0]
		userID = strings.Split(userID, ":")[1]
	case strings.HasPrefix(resource, "https://"+domain+"/"): // https://${Domain}/${User}
		userID = strings.Split(resource, domain+"/")[1]
	default:
		w.WriteHeader(400)
		return
	}

	result := bytes.Replace(webfingerTemplate, []byte("${Resource}"), []byte(fmt.Sprintf("acct:%s@%s", userID, domain)), 1)
	result = bytes.ReplaceAll(result, []byte("${User}"), []byte(userID))

	w.Write(result)
}
