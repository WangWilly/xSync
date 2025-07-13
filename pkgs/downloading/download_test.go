package downloading

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/WangWilly/xSync/pkgs/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/database"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/pkgs/downloading/heaphelper"
	"github.com/WangWilly/xSync/pkgs/model"
	"github.com/WangWilly/xSync/pkgs/utils"
	"github.com/jmoiron/sqlx"
)

var db *sqlx.DB

/*
创建
改名
*/
func init() {
	var err error
	path := filepath.Join(os.TempDir(), "test.db")
	err = os.RemoveAll(path)
	if err != nil {
		panic(err)
	}

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&cache=shared", path)
	db, err = sqlx.Connect("sqlite3", dsn)
	if err != nil {
		panic(err)
	}
	model.CreateTables(db)
}

func TestUserEntity(t *testing.T) {
	tempdir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Error(err)
		return
	}

	name := "test"
	uid := 0

	os.RemoveAll(filepath.Join(tempdir, name))
	testSyncUser(t, name, uid, tempdir, false)

	// 改名
	name = name + "renamed"
	os.RemoveAll(filepath.Join(tempdir, name))
	testSyncUser(t, name, uid, tempdir, true)

	// 什么都不干
	os.RemoveAll(filepath.Join(tempdir, name))
	ue := testSyncUser(t, name, uid, tempdir, true)

	if !ue.LatestReleaseTime().IsZero() {
		t.Errorf("default time is not null")
		return
	}

	now := time.Now()
	if err := ue.SetLatestReleaseTime(now); err != nil {
		t.Error(err)
		return
	}
	if !ue.LatestReleaseTime().Equal(now) {
		t.Errorf("latest release: %v, want %v", ue.LatestReleaseTime(), now)
	}

	record, err := database.GetUserEntityById(db, ue.Id())
	if err != nil {
		t.Error(err)
		return
	}
	if !record.LatestReleaseTime.Time.Equal(now) {
		t.Errorf("recorded time: %v, want %v", record.LatestReleaseTime.Time, now)
	}

	// remove
	eid := ue.Id()
	if err := ue.Remove(); err != nil {
		t.Error(err)
		return
	}

	ex, err := utils.PathExists(filepath.Join(tempdir, name))
	if err != nil {
		t.Error(err)
		return
	}
	if ex {
		t.Errorf("dir is exist after remove")
	}

	record, err = database.GetUserEntityById(db, eid)
	if err != nil {
		t.Error(err)
		return
	}
	if record != nil {
		t.Errorf("record is exist after remove")
	}
}

func TestListEntity(t *testing.T) {
	tempdir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Error(err)
		return
	}

	name := "test"
	uid := 0
	os.RemoveAll(filepath.Join(tempdir, name))
	testSyncList(t, name, uid, tempdir, false)

	// 改名
	name = name + "renamed"
	os.RemoveAll(filepath.Join(tempdir, name))
	testSyncList(t, name, uid, tempdir, true)

	os.RemoveAll(filepath.Join(tempdir, name))
	le := testSyncList(t, name, uid, tempdir, true)

	// remove
	eid := le.Id()
	if err := le.Remove(); err != nil {
		t.Error(err)
		return
	}

	ex, err := utils.PathExists(filepath.Join(tempdir, name))
	if err != nil {
		t.Error(err)
		return
	}
	if ex {
		t.Errorf("dir is exist after remove")
	}

	record, err := database.GetListEntityById(db, eid)
	if err != nil {
		t.Error(err)
		return
	}
	if record != nil {
		t.Errorf("record is exist after remove")
	}
}

func verifyDir(t *testing.T, entity smartpathdto.SmartPath, wantPath string) {
	path, _ := entity.Path()
	if wantPath != path {
		t.Errorf("path: %s, want %s", path, wantPath)
		return
	}

	// 目录存在
	ex, err := utils.PathExists(path)
	if err != nil {
		t.Error(err)
		return
	}
	if !ex {
		t.Errorf("path is not exist")
		return
	}
}

func verifyUserRecord(t *testing.T, entity smartpathdto.SmartPath, uid uint64, name string, parentDir string) *smartpathdto.UserSmartPath {
	wantPath := filepath.Join(parentDir, name)
	record, err := database.GetUserEntity(db, uid, parentDir)
	if err != nil {
		t.Error(err)
		return nil
	}
	if int(record.Id.Int32) != entity.Id() {
		t.Errorf("eid: %d, want %d", entity.Id(), record.Id.Int32)
	}
	if record.Path() != wantPath {
		t.Errorf("recorded path: %s, want %s", record.Path(), wantPath)
	}
	if record.Name != name {
		t.Errorf("recorded name: %s, want %s", record.Name, name)
	}
	if record.Uid != uid {
		t.Errorf("uid: %d, want %d", record.Uid, uid)
	}
	return entity.(*smartpathdto.UserSmartPath)
}

func verifyLstRecord(t *testing.T, entity smartpathdto.SmartPath, lid int64, name string, parentDir string) {
	wantPath := filepath.Join(parentDir, name)
	record, err := database.GetListEntity(db, lid, parentDir)
	if err != nil {
		t.Error(err)
		return
	}
	if int(record.Id.Int32) != entity.Id() {
		t.Errorf("eid: %d, want %d", entity.Id(), record.Id.Int32)
	}
	if record.Path() != wantPath {
		t.Errorf("recorded path: %s, want %s", record.Path(), wantPath)
	}
	if record.Name != name {
		t.Errorf("recorded name: %s, want %s", record.Name, name)
	}
	if record.LstId != lid {
		t.Errorf("uid: %d, want %d", record.LstId, lid)
	}
}

func testSyncUser(t *testing.T, name string, uid int, parentdir string, exist bool) *smartpathdto.UserSmartPath {
	ue, err := smartpathdto.NewUserSmartPath(db, uint64(uid), parentdir)
	if err != nil {
		t.Error(err)
		return nil
	}

	// 创建状态正确
	if ue.IsSyncToDb() && !exist {
		t.Errorf("ue.Created = true, want false")
	} else if !ue.IsSyncToDb() && exist {
		t.Errorf("ue.created = false, want true")
	}

	if err := heaphelper.ExpectNameMustExistOnStorage(ue, name); err != nil {
		t.Error(err)
		return nil
	}

	// 测试同步后路径
	wantPath := filepath.Join(parentdir, name)
	verifyDir(t, ue, wantPath)

	// 记录正确
	verifyUserRecord(t, ue, uint64(uid), name, parentdir)

	if ue.TwitterId() != uint64(uid) {
		t.Errorf("uid: %d, want %d", ue.TwitterId(), uid)
	}
	return ue
}

func testSyncList(t *testing.T, name string, lid int, parentDir string, exist bool) *smartpathdto.ListSmartPath {
	le, err := smartpathdto.NewListSmartPath(db, int64(lid), parentDir)
	if err != nil {
		t.Error(err)
		return nil
	}

	// 创建状态正确
	if le.Created && !exist {
		t.Errorf("ue.Created = true, want false")
	} else if !le.Created && exist {
		t.Errorf("ue.created = false, want true")
	}

	if err := heaphelper.ExpectNameMustExistOnStorage(le, name); err != nil {
		t.Error(err)
		return nil
	}

	// 测试同步后路径
	wantPath := filepath.Join(parentDir, name)
	verifyDir(t, le, wantPath)

	// 记录正确
	verifyLstRecord(t, le, int64(lid), name, parentDir)
	return le
}

func TestDumper(t *testing.T) {
	dumper := NewDumper()

	n := 3
	tweets := generateSomeTweets(n * 10)

	k := 0
	for i := 0; i < n; i++ {
		for j := 0; j < 10; j++ {
			dumper.Push(i, tweets[k])
			k++
		}

	}

	// 重复推送
	k = 0
	for i := 0; i < n; i++ {
		for j := 0; j < 10; j++ {
			if dumper.Push(i, tweets[k]) != 0 {
				t.Errorf("repeat push")
			}
			k++
		}

	}

	if dumper.count != 30 {
		t.Errorf("dumper.count: %d, want %d", dumper.count, 30)
	}

	f, err := os.CreateTemp("", "")
	if err != nil {
		t.Error(err)
		return
	}

	if err := dumper.Dump(f.Name()); err != nil {
		t.Error(err)
		return
	}

	dumper.Clear()
	if dumper.count != 0 {
		t.Errorf("dumper.count: %d, want %d", dumper.count, 0)
		return
	}

	if err := dumper.Load(f.Name()); err != nil {
		t.Error(err)
		return
	}

	if dumper.count != 30 {
		t.Errorf("dumper.count: %d want %d", dumper.count, 30)
	}

	k = 0
	for i := 0; i < n; i++ {
		for j := 0; j < 10; j++ {
			if dumper.Push(i, tweets[k]) != 0 {
				t.Errorf("repeat push after load")
				break
			}
			k++
		}

	}
}

func generateSomeTweets(n int) []*twitterclient.Tweet {
	res := []*twitterclient.Tweet{}
	for i := 0; i < n; i++ {
		tw := &twitterclient.Tweet{}
		tw.CreatedAt = time.Now()
		tw.Creator = nil
		tw.Id = uint64(i)
		tw.Text = fmt.Sprintf("tweet %d", i)
		res = append(res, tw)
	}
	return res
}
