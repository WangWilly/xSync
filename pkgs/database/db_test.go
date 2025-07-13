package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/WangWilly/xSync/pkgs/model"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var db *sqlx.DB

func opentmpdb() *sqlx.DB {
	var err error
	tmpFile, err := os.CreateTemp("", "")
	if err != nil {
		panic(err)
	}
	path := tmpFile.Name()

	db, err = sqlx.Connect("sqlite3", fmt.Sprintf("file:%s?_journal_mode=WAL&cache=shared", path))

	if err != nil {
		panic(err)
	}

	model.CreateTables(db)
	return db
}

func generateUser(n int) *model.User {
	usr := &model.User{}
	usr.Id = uint64(n)
	name := fmt.Sprintf("user%d", n)
	usr.ScreenName = name
	usr.Name = name
	return usr
}
func TestUserOperation(t *testing.T) {
	db = opentmpdb()
	defer db.Close()

	n := 100
	users := make([]*model.User, n)
	for i := 0; i < n; i++ {
		users[i] = generateUser(i)
	}

	for _, usr := range users {
		testUser(t, usr)
	}
}

func testUser(t *testing.T, usr *model.User) {
	// create
	if err := CreateUser(db, usr); err != nil {
		t.Error(err)
		return
	}

	same, err := hasSameUserRecord(usr)
	if err != nil {
		t.Error(err)
		return
	}
	if !same {
		t.Error("record mismatch after create user")
		return
	}

	// update
	usr.Name = "renamed"
	if err := UpdateUser(db, usr); err != nil {
		t.Error(err)
		return
	}

	same, err = hasSameUserRecord(usr)
	if err != nil {
		t.Error(err)
		return
	}
	if !same {
		t.Error("record mismatch after update user")
		return
	}

	// record previous name
	if err = RecordUserPreviousName(db, usr.Id, usr.Name, usr.ScreenName); err != nil {
		t.Error(err)
		return
	}

	// delete
	if err = DelUser(db, usr.Id); err != nil {
		t.Error(err)
		return
	}

	usr, err = GetUserById(db, usr.Id)
	if err != nil {
		t.Error(err)
		return
	}
	if usr != nil {
		t.Error("record mismatch after delete user")
	}
}

func hasSameUserRecord(usr *model.User) (bool, error) {
	retrieved, err := GetUserById(db, usr.Id)
	if err != nil {
		return false, err
	}
	if retrieved == nil {
		return false, nil
	}
	// Compare only the main fields, not the timestamps
	return retrieved.Id == usr.Id &&
		retrieved.ScreenName == usr.ScreenName &&
		retrieved.Name == usr.Name &&
		retrieved.IsProtected == usr.IsProtected &&
		retrieved.FriendsCount == usr.FriendsCount, nil
}

func generateList(id int) *model.List {
	lst := &model.List{}
	lst.Id = uint64(id)
	lst.Name = fmt.Sprintf("lst%d", id)
	return lst
}

func TestList(t *testing.T) {
	db = opentmpdb()
	defer db.Close()
	n := 100
	lsts := make([]*model.List, n)
	for i := 0; i < n; i++ {
		lsts[i] = generateList(i)
	}

	for _, lst := range lsts {
		// create
		if err := CreateLst(db, lst); err != nil {
			t.Error(err)
			return
		}

		// read
		same, err := isSameLstRecord(lst)
		if err != nil {
			t.Error(err)
			return
		}
		if !same {
			t.Error("record mismatch after create lst")
			return
		}

		// update
		lst.Name = "renamed"
		if err = UpdateLst(db, lst); err != nil {
			t.Error(err)
			return
		}
		same, err = isSameLstRecord(lst)
		if err != nil {
			t.Error(err)
			return
		}
		if !same {
			t.Error("record mismatch after update lst")
			return
		}

		// delete
		if err = DelLst(db, lst.Id); err != nil {
			t.Error(err)
			return
		}
		record, err := GetLst(db, lst.Id)
		if err != nil {
			t.Error(err)
			return
		}
		if record != nil {
			t.Error("record mismatch after delete lst")
			return
		}
	}
}

func isSameLstRecord(lst *model.List) (bool, error) {
	record, err := GetLst(db, lst.Id)
	if err != nil {
		return false, err
	}
	if record == nil {
		return false, nil
	}
	// Compare only the main fields, not the timestamps
	return record.Id == lst.Id &&
		record.Name == lst.Name &&
		record.OwnerId == lst.OwnerId, nil
}

func TestUserEntity(t *testing.T) {
	db = opentmpdb()
	defer db.Close()
	n := 100
	entities := make([]*model.UserEntity, n)
	tempDir := os.TempDir()
	for i := 0; i < n; i++ {
		entities[i] = generateUserEntity(uint64(i), tempDir)
	}

	for _, entity := range entities {
		// path
		expectedPath := filepath.Join(tempDir, entity.Name)
		if expectedPath != entity.Path() {
			t.Errorf("entity.Path() = %v want %v", entity.Path(), expectedPath)
			return
		}

		// create
		if err := CreateUserEntity(db, entity); err != nil {
			t.Error(err)
			return
		}

		// read
		yes, err := hasSameUserEntityRecord(entity)
		if err != nil {
			t.Error(err)
			return
		}
		if !yes {
			t.Error("record mismatch after create user entity")
			return
		}

		// update
		entity.Name = entity.Name + "renamed"
		if err := UpdateUserEntity(db, entity); err != nil {
			t.Error(err)
			return
		}
		yes, err = hasSameUserEntityRecord(entity)
		if err != nil {
			t.Error(err)
			return
		}
		if !yes {
			t.Error("record mismatch after update user entity")
			return
		}

		// latest release time
		now := time.Now()
		if err = UpdateUserEntityTweetStat(db, int(entity.Id.Int32), now, 25); err != nil {
			t.Error(err)
			return
		}
		entity.MediaCount.Scan(25)

		// locate
		record, err := GetUserEntity(db, entity.Uid, entity.ParentDir)
		if err != nil {
			t.Error(err)
			return
		}
		if record == nil {
			t.Error("record mismatch on locate user entity")
			return
		}
		// 单独比较时间字段
		if !record.LatestReleaseTime.Time.Equal(now) {
			t.Errorf("recorded latest release time: %v want %v", record.LatestReleaseTime.Time, now)
		}
		// Compare only the main fields, not the timestamps
		if record.Id != entity.Id ||
			record.Uid != entity.Uid ||
			record.Name != entity.Name ||
			record.ParentDir != entity.ParentDir ||
			record.MediaCount != entity.MediaCount {
			t.Error("record mismatch on locate user entity")
			return
		}

		// delete
		if err = DelUserEntity(db, uint32(entity.Id.Int32)); err != nil {
			t.Error(err)
			return
		}

		yes, err = hasSameUserEntityRecord(entity)
		if err != nil {
			t.Error(err)
			return
		}
		if yes {
			t.Error("record mismatch after delete user entity")
		}
	}
}

func generateUserEntity(uid uint64, pdir string) *model.UserEntity {
	ue := model.UserEntity{}
	user := generateUser(int(uid))
	if err := CreateUser(db, user); err != nil {
		panic(err)
	}

	ue.Name = user.Name
	ue.Uid = uid
	ue.ParentDir = pdir
	return &ue
}

func hasSameUserEntityRecord(entity *model.UserEntity) (bool, error) {
	record, err := GetUserEntityById(db, int(entity.Id.Int32))
	if err != nil {
		return false, err
	}
	if record == nil {
		return false, nil
	}
	// Compare only the main fields, not the timestamps
	return record.Id == entity.Id &&
		record.Uid == entity.Uid &&
		record.Name == entity.Name &&
		record.ParentDir == entity.ParentDir &&
		record.MediaCount == entity.MediaCount, nil
}

func TestLstEntity(t *testing.T) {
	db = opentmpdb()
	defer db.Close()
	tempdir := os.TempDir()
	n := 100
	entities := make([]*model.ListEntity, n)
	for i := 0; i < n; i++ {
		entities[i] = generateLstEntity(int64(i), tempdir)
	}

	for _, entity := range entities {
		// path
		expectedPath := filepath.Join(tempdir, entity.Name)
		if expectedPath != entity.Path() {
			t.Errorf("entity.Path() = %v want %v", entity.Path(), expectedPath)
			return
		}
		// create
		if err := CreateLstEntity(db, entity); err != nil {
			t.Error(err)
			return
		}

		// read
		yes, err := hasSameLstEntityRecord(entity)
		if err != nil {
			t.Error(err)
			return
		}
		if !yes {
			t.Error("record mismatch after create lst entity")
		}

		// update
		entity.Name = entity.Name + "renamed"
		if err = UpdateLstEntity(db, entity); err != nil {
			t.Error(err)
			return
		}
		yes, err = hasSameLstEntityRecord(entity)
		if err != nil {
			t.Error(err)
			return
		}
		if !yes {
			t.Error("record mismatch after update lst entity")
			return
		}

		// locate
		record, err := GetListEntity(db, entity.LstId, entity.ParentDir)
		if err != nil {
			t.Error(err)
			return
		}
		if record == nil {
			t.Error("record mismatch after locate lst entity")
			return
		}
		// Compare only the main fields, not the timestamps
		if record.Id != entity.Id ||
			record.LstId != entity.LstId ||
			record.Name != entity.Name ||
			record.ParentDir != entity.ParentDir {
			t.Error("record mismatch after locate lst entity")
			return
		}

		// delete
		if err = DelLstEntity(db, int(entity.Id.Int32)); err != nil {
			t.Error(err)
			return
		}
		yes, err = hasSameLstEntityRecord(entity)
		if err != nil {
			t.Error(err)
			return
		}
		if yes {
			t.Error("record mismatch after delete lst entity")
			return
		}
	}
}

func generateLstEntity(lid int64, pdir string) *model.ListEntity {
	lst := generateList(int(lid))
	if err := CreateLst(db, lst); err != nil {
		panic(err)
	}
	entity := model.ListEntity{}
	entity.LstId = lid
	entity.ParentDir = pdir
	entity.Name = lst.Name
	return &entity
}

func hasSameLstEntityRecord(entity *model.ListEntity) (bool, error) {
	record, err := GetListEntityById(db, int(entity.Id.Int32))
	if err != nil {
		return false, err
	}
	if record == nil {
		return false, nil
	}
	// Compare only the main fields, not the timestamps
	return record.Id == entity.Id &&
		record.LstId == entity.LstId &&
		record.Name == entity.Name &&
		record.ParentDir == entity.ParentDir, nil
}

func TestLink(t *testing.T) {
	db = opentmpdb()
	defer db.Close()
	n := 100
	links := make([]*model.UserLink, n)
	for i := 0; i < n; i++ {
		links[i] = generateLink(i, i)
	}

	for _, link := range links {
		// path
		le, err := GetListEntityById(db, int(link.ListEntityIdBelongTo))
		if err != nil {
			t.Error(err)
			return
		}
		expectedPath := filepath.Join(le.Path(), link.Name)
		path, err := link.Path(db)
		if err != nil {
			t.Error(err)
			return
		}
		if expectedPath != path {
			t.Errorf("link.Path() = %v want %v", path, expectedPath)
			return
		}

		// c
		if err := CreateUserLink(db, link); err != nil {
			t.Error(err)
			return
		}

		// r
		yes, err := hasSameUserLinkRecord(link)
		if err != nil {
			t.Error(err)
			return
		}
		if !yes {
			t.Error("mismatch record after create user link")
			return
		}

		records, err := GetUserLinks(db, link.UserTwitterId)
		if err != nil {
			t.Error(err)
			return
		}
		if len(records) != 1 {
			t.Error("mismatch record after get all user links")
			return
		}
		// Compare only the main fields, not the timestamps
		record := records[0]
		if record.Id != link.Id ||
			record.UserTwitterId != link.UserTwitterId ||
			record.Name != link.Name ||
			record.ListEntityIdBelongTo != link.ListEntityIdBelongTo {
			t.Error("mismatch record after get all user links")
			return
		}

		// u
		link.Name = link.Name + "renamed"
		if err = UpdateUserLink(db, link.Id.Int32, link.Name); err != nil {
			t.Error(err)
			return
		}
		yes, err = hasSameUserLinkRecord(link)
		if err != nil {
			t.Error(err)
			return
		}
		if !yes {
			t.Error("mismatch record after update user link")
			return
		}

		// d
		if err := DelUserLink(db, link.Id.Int32); err != nil {
			t.Error(err)
			return
		}
		yes, err = hasSameUserLinkRecord(link)
		if err != nil {
			t.Error(err)
			return
		}
		if yes {
			t.Error("mismatch record after delete user link")
			return
		}
	}
}

func generateLink(uid int, lid int) *model.UserLink {
	usr := generateUser(uid)
	le := generateLstEntity(int64(lid), os.TempDir())
	if err := CreateLstEntity(db, le); err != nil {
		panic(err)
	}

	ul := model.UserLink{}
	ul.Name = fmt.Sprintf("%d-%d", lid, uid)
	ul.ListEntityIdBelongTo = le.Id.Int32
	ul.UserTwitterId = usr.Id
	return &ul
}

func hasSameUserLinkRecord(link *model.UserLink) (bool, error) {
	record, err := GetUserLink(db, link.UserTwitterId, link.ListEntityIdBelongTo)
	if err != nil {
		return false, err
	}
	if record == nil {
		return false, nil
	}
	// Compare only the main fields, not the timestamps
	return record.Id == link.Id &&
		record.UserTwitterId == link.UserTwitterId &&
		record.Name == link.Name &&
		record.ListEntityIdBelongTo == link.ListEntityIdBelongTo, nil
}

func benchmarkUpdateUser(b *testing.B, routines int) {
	db = opentmpdb()
	defer db.Close()

	n := 500
	users := make(chan *model.User, n)
	for i := 0; i < n; i++ {
		user := generateUser(i)
		if err := CreateUser(db, user); err != nil {
			b.Error(err)
			return
		}
		user.Name = user.Name + "renamed"
		users <- user
	}
	close(users)

	wg := sync.WaitGroup{}
	routine := func() {
		defer wg.Done()
		for user := range users {
			if err := UpdateUser(db, user); err != nil {
				b.Error(err)
			}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < routines; j++ {
			wg.Add(1)
			go routine()
		}
		wg.Wait()
	}
}

func BenchmarkUpdateUser1(b *testing.B) {
	benchmarkUpdateUser(b, 1)
}

func BenchmarkUpdateUser6(b *testing.B) {
	benchmarkUpdateUser(b, 6)
}

func BenchmarkUpdateUser12(b *testing.B) {
	benchmarkUpdateUser(b, 12)
}

func BenchmarkUpdateUser24(b *testing.B) {
	benchmarkUpdateUser(b, 24)
}
