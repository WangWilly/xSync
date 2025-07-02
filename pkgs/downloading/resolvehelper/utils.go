package resolvehelper

import (
	"os"

	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/pkgs/twitter"
)

// TODO: make private
func SyncPath(path smartpathdto.SmartPath, expectedName string) error {
	if !path.Recorded() {
		return path.Create(expectedName)
	}

	if path.Name() != expectedName {
		return path.Rename(expectedName)
	}

	p, err := path.Path()
	if err != nil {
		return err
	}

	return os.MkdirAll(p, 0755)
}

// TODO: make private
// IsIngoreUser checks if a user should be ignored during processing
func IsIngoreUser(user *twitter.User) bool {
	return user.Blocking || user.Muting
}
