package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	dbFileName  = ".kv-recon.db"
	tokenBucket = "tokens"
)

// TokenEntry is a persisted token record.
type TokenEntry struct {
	ID           string    `json:"id"`
	Label        string    `json:"label"`
	TenantID     string    `json:"tenantId"`
	ClientID     string    `json:"clientId"`
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
	Scopes       string    `json:"scopes"`
}

var db *bolt.DB

// openDB opens (or creates) ~/.kv-recon.db and ensures the tokens bucket exists.
func openDB() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	path := filepath.Join(home, dbFileName)

	db, err = bolt.Open(path, 0600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return fmt.Errorf("open db %s: %w", path, err)
	}

	return db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(tokenBucket))
		return err
	})
}

// SaveToken writes (or overwrites) a TokenEntry keyed by its ID.
// If t.ID is empty a random 8-byte hex ID is generated automatically.
func SaveToken(t TokenEntry) error {
	if t.ID == "" {
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return fmt.Errorf("generate id: %w", err)
		}
		t.ID = hex.EncodeToString(b)
	}
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(tokenBucket)).Put([]byte(t.ID), data)
	})
}

// GetToken retrieves a single TokenEntry by ID.
func GetToken(id string) (TokenEntry, error) {
	var t TokenEntry
	err := db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket([]byte(tokenBucket)).Get([]byte(id))
		if v == nil {
			return fmt.Errorf("token %q not found", id)
		}
		return json.Unmarshal(v, &t)
	})
	return t, err
}

// ListTokens returns all stored TokenEntry records.
func ListTokens() ([]TokenEntry, error) {
	var tokens []TokenEntry
	err := db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(tokenBucket)).ForEach(func(_, v []byte) error {
			var t TokenEntry
			if err := json.Unmarshal(v, &t); err != nil {
				return err
			}
			tokens = append(tokens, t)
			return nil
		})
	})
	return tokens, err
}

// DeleteToken removes a token by ID.
func DeleteToken(id string) error {
	return db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(tokenBucket)).Delete([]byte(id))
	})
}
