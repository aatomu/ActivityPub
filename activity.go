package main

import (
	"os"
	"path/filepath"
)

func getNote(userID, noteID string) (note []byte, err error) {
	return os.ReadFile(filepath.Join("./users", userID, "note", noteID+".json"))
}

func getAttachment(userID, attachment string) (attachmentData []byte, err error) {
	return os.ReadFile(filepath.Join("./users", userID, "attachment", attachment))
}
