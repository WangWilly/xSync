package downloading

import (
	"github.com/WangWilly/xSync/pkgs/database"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/pkgs/twitter"
	"github.com/WangWilly/xSync/pkgs/utils"
	"github.com/jmoiron/sqlx"
)

////////////////////////////////////////////////////////////////////////////////
// User and Entity Synchronization
////////////////////////////////////////////////////////////////////////////////

// syncTwitterUserToDb updates the database record for a user
// 更新数据库中对用户的记录
func syncTwitterUserToDb(db *sqlx.DB, twitterUser *twitter.User) error {
	renamed := false
	isNew := false
	userRecord, err := database.GetUserById(db, twitterUser.TwitterId)
	if err != nil {
		return err
	}

	if userRecord == nil {
		isNew = true
		userRecord = &database.User{}
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

func syncUserToDbAndGetSmartPath(db *sqlx.DB, user *twitter.User, dir string) (*smartpathdto.UserSmartPath, error) {
	if err := syncTwitterUserToDb(db, user); err != nil {
		return nil, err
	}
	expectedFileName := utils.ToLegalWindowsFileName(user.Title())

	userSmartPath, err := smartpathdto.NewUserSmartPath(db, user.TwitterId, dir)
	if err != nil {
		return nil, err
	}
	if err = syncPath(userSmartPath, expectedFileName); err != nil {
		return nil, err
	}
	return userSmartPath, nil
}

////////////////////////////////////////////////////////////////////////////////
// Utility Functions
////////////////////////////////////////////////////////////////////////////////

// calcUserDepth calculates how many timeline requests are needed to get all user tweets
// 需要请求多少次时间线才能获取完毕用户的推文？
func calcUserDepth(exist int, total int) int {
	if exist >= total {
		return 1
	}

	miss := total - exist
	depth := miss / twitter.AvgTweetsPerPage
	if miss%twitter.AvgTweetsPerPage != 0 {
		depth++
	}
	if exist == 0 {
		depth++ //对于新用户，需要多获取一个空页
	}
	return depth
}

// shouldIngoreUser checks if a user should be ignored during processing
func shouldIngoreUser(user *twitter.User) bool {
	return user.Blocking || user.Muting
}
