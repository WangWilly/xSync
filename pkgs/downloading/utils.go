package downloading

import (
	"os"
	"path/filepath"

	"github.com/WangWilly/xSync/pkgs/database"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/jmoiron/sqlx"
)

////////////////////////////////////////////////////////////////////////////////

func syncPath(path smartpathdto.SmartPath, expectedName string) error {
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

////////////////////////////////////////////////////////////////////////////////

func updateUserLink(lnk *database.UserLink, db *sqlx.DB, path string) error {
	name := filepath.Base(path)

	linkpath, err := lnk.Path(db)
	if err != nil {
		return err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return err
	}

	if lnk.Name == name {
		// 用户未改名，但仍应确保链接存在
		err = os.Symlink(path, linkpath)
		if os.IsExist(err) {
			err = nil
		}
		return err
	}

	newlinkpath := filepath.Join(filepath.Dir(linkpath), name)

	if err = os.RemoveAll(linkpath); err != nil {
		return err
	}
	if err = os.Symlink(path, newlinkpath); err != nil && !os.IsExist(err) {
		return err
	}

	if err = database.UpdateUserLink(db, lnk.Id.Int32, name); err != nil {
		return err
	}

	lnk.Name = name
	return nil
}
