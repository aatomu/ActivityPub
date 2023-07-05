package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func GetFollower(userID string) (follower ActivityStreamOrderedCollection, err error) {
	followerFile := filepath.Join("./users", userID, "follower.json")

	followerList, err := os.ReadFile(followerFile)
	if err != nil {
		return follower, err
	}
	err = json.Unmarshal(followerList, &follower)
	if err != nil {
		return follower, err
	}
	return follower, nil
}

func SaveFollower(userID string, follower ActivityStreamOrderedCollection) error {
	followerFile := filepath.Join("./users", userID, "follower.json")

	followerList, _ := json.MarshalIndent(follower, "", "  ")
	err := os.WriteFile(followerFile, followerList, 0666)
	if err != nil {
		return err
	}

	return nil
}
