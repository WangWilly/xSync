package storage

import (
	"os"
	"path/filepath"
)

////////////////////////////////////////////////////////////////////////////////
// Storage Path Management Structure
////////////////////////////////////////////////////////////////////////////////

// StorePath represents the application's storage paths
type StorePath struct {
	Root   string
	Users  string
	Data   string
	DB     string
	ErrorJ string
}

////////////////////////////////////////////////////////////////////////////////
// Storage Path Management Functions
////////////////////////////////////////////////////////////////////////////////

// NewStorePath creates a new StorePath instance and ensures directories exist
func NewStorePath(root string) (*StorePath, error) {
	ph := StorePath{}
	ph.Root = root
	ph.Users = filepath.Join(root, "users")
	ph.Data = filepath.Join(root, ".data")

	ph.DB = filepath.Join(ph.Data, "foo.db")
	ph.ErrorJ = filepath.Join(ph.Data, "errors.json")

	// ensure folders exist
	err := os.Mkdir(ph.Root, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}

	err = os.Mkdir(ph.Users, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}

	err = os.Mkdir(ph.Data, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}
	return &ph, nil
}
