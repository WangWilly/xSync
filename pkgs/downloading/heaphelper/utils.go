package heaphelper

import (
	"os"
	"path/filepath"

	"github.com/WangWilly/xSync/pkgs/clipkg/database"
	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/jmoiron/sqlx"
)

// isIngoreUser checks if a user should be ignored during processing
func isIngoreUser(user *twitterclient.User) bool {
	return user.Blocking || user.Muting
}

func syncUserToDbAndGetSmartPath(db *sqlx.DB, user *twitterclient.User, dir string) (*smartpathdto.UserSmartPath, error) {
	if err := syncTwitterUserToDb(db, user); err != nil {
		return nil, err
	}
	expectedFileName := utils.ToLegalWindowsFileName(user.Title())

	userSmartPath, err := smartpathdto.NewUserSmartPath(db, user.TwitterId, dir)
	if err != nil {
		return nil, err
	}
	if err = ExpectNameMustExistOnStorage(userSmartPath, expectedFileName); err != nil {
		return nil, err
	}
	return userSmartPath, nil
}

func ExpectNameMustExistOnStorage(path smartpathdto.SmartPath, expectedName string) error {
	if !path.IsSyncToDb() {
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

// syncTwitterUserToDb updates the database record for a user
// 更新数据库中对用户的记录
func syncTwitterUserToDb(db *sqlx.DB, twitterUser *twitterclient.User) error {
	renamed := false
	isNew := false
	userRecord, err := database.GetUserById(db, twitterUser.TwitterId)
	if err != nil {
		return err
	}

	if userRecord == nil {
		isNew = true
		userRecord = &model.User{}
		userRecord.Id = twitterUser.TwitterId
	} else {
		renamed = userRecord.Name != twitterUser.Name || userRecord.ScreenName != twitterUser.ScreenName
	}

	userRecord.FriendsCount = twitterUser.FriendsCount
	userRecord.IsProtected = twitterUser.IsProtected
	userRecord.Name = twitterUser.Name
	userRecord.ScreenName = twitterUser.ScreenName

	if isNew {
		err = database.CreateUser(db, userRecord)
	} else {
		err = database.UpdateUser(db, userRecord)
	}
	if err != nil {
		return err
	}
	if renamed || isNew {
		err = database.RecordUserPreviousName(db, twitterUser.TwitterId, twitterUser.Name, twitterUser.ScreenName)
	}
	return err
}

////////////////////////////////////////////////////////////////////////////////

func updateUserLink(lnk *model.UserLink, db *sqlx.DB, path string) error {
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

////////////////////////////////////////////////////////////////////////////////

// calcUserDepth calculates how many timeline requests are needed to get all user tweets
// 需要请求多少次时间线才能获取完毕用户的推文？
func calcUserDepth(exist int, total int) int {
	if exist >= total {
		return 1
	}

	miss := total - exist
	depth := miss / twitterclient.AvgTweetsPerPage
	if miss%twitterclient.AvgTweetsPerPage != 0 {
		depth++
	}
	if exist == 0 {
		depth++ //对于新用户，需要多获取一个空页
	}
	return depth
}
