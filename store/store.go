// Package store provides CRUD (Create, Read, Update, Delet) operations for a simple data store encoded in a json file.
package store

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
)

type store struct {
	Application string                 `json:"Application"`
	Keys        map[string]interface{} `json:"Keys"`
}

// ErrNotFound is returned when the requested key is not found in the store.
var ErrNotFound = errors.New("store: key not found")

// Create generates a new store if one doesn't already exist and inserts the key and value into it.
// If a key already exists, Create acts as an Update to the store.
// Returns any write error encountered.
func Create(key string, value interface{}) error {
	// Check if file exists first
	err := createFileIfNotExist()
	if err != nil {
		return err
	}

	// Read store into struct.
	store, err := readFromStore()
	if err != nil {
		return err
	}

	// Insert new value into the map.
	store.Keys[key] = value

	// Add back into the file
	writeToStore(*store)

	return nil
}

// Read gets a value associated with a given key in the store.
// Returns a pointer to the associated value and any read error encountered.
func Read(key string) (*interface{}, error) {
	// Read store into struct.
	store, err := readFromStore()
	if err != nil {
		return nil, err
	}

	// Check if the key exists.
	if store.Keys[key] == nil {
		return nil, ErrNotFound
	}

	// Return corresponding value.
	value := store.Keys[key]
	return &value, nil
}

// Update sets a new value for a key that already exists in the store.
// Returns any write error encountered.
func Update(key string, value interface{}) error {
	// Read store into struct.
	store, err := readFromStore()
	if err != nil {
		return err
	}

	// Check if the key exists.
	if store.Keys[key] == nil {
		return ErrNotFound
	}

	// Insert updated value into struct.
	store.Keys[key] = value

	// Write back into the file.
	writeToStore(*store)

	return nil
}

// Delete removes the key-value pair for the given key.
// Returns any read or write error encountered.
func Delete(key string) error {
	// Read store into struct.
	store, err := readFromStore()
	if err != nil {
		return err
	}

	// Check if the key exists.
	if store.Keys[key] == nil {
		return ErrNotFound
	}

	// Delete key and value.
	delete(store.Keys, key)

	// Write back into the file.
	writeToStore(*store)

	return nil
}

// DeleteAll clears all key-value pairs in the store.
// Returns any write error encountered.
func DeleteAll() error {
	//Overwrite the old store
	return writeNewStore()
}

// DeleteStore deletes the file that contains the store.
// Returns any remove error encountered.
func DeleteStore() error {
	return os.Remove("./store.json")
}

func createFileIfNotExist() error {
	// Check that the file doesn't exist.
	if _, err := os.Stat("./store.json"); os.IsNotExist(err) {
		return writeNewStore()
	}

	return nil
}

func writeNewStore() error {
	b := store{
		Application: "spotify-cli",
		Keys:        map[string]interface{}{},
	}

	return writeToStore(b)
}

func writeToStore(storeStruct store) error {
	// Serialise the struct as json.
	storeEnc, err := json.Marshal(storeStruct)
	if err != nil {
		return err
	}

	// Write to file.
	err = ioutil.WriteFile("./store.json", storeEnc, 0640)
	if err != nil {
		return err
	}

	return nil
}

func readFromStore() (*store, error) {
	// Read file.
	data, err := ioutil.ReadFile("./store.json")
	if err != nil {
		return nil, err
	}

	// Unmarshal data.
	store := store{}
	err = json.Unmarshal(data, &store)
	if err != nil {
		return nil, err
	}

	return &store, nil
}