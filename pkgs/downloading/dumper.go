package downloading

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/userentityrepo"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/dldto"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/jmoiron/sqlx"
)

type TweetDumper struct {
	db             *sqlx.DB
	userEntityRepo UserEntityRepo

	userTweetsMap   map[int][]*twitterclient.Tweet
	usersTweetIdSet map[int]map[uint64]struct{}
	count           int
}

func NewDumper(db *sqlx.DB) *TweetDumper {
	td := TweetDumper{
		db:              db,
		userEntityRepo:  userentityrepo.New(),
		userTweetsMap:   make(map[int][]*twitterclient.Tweet),
		usersTweetIdSet: make(map[int]map[uint64]struct{}),
	}
	return &td
}

////////////////////////////////////////////////////////////////////////////////

func (td *TweetDumper) Push(
	userEntityId int,
	tweets ...*twitterclient.Tweet,
) int {
	_, ok := td.userTweetsMap[userEntityId]
	if !ok {
		td.userTweetsMap[userEntityId] = make([]*twitterclient.Tweet, 0, len(tweets))
		td.usersTweetIdSet[userEntityId] = make(map[uint64]struct{})
	}

	oldCount := td.count

	for _, tw := range tweets {
		_, exist := td.usersTweetIdSet[userEntityId][tw.Id]
		if exist {
			continue
		}
		td.userTweetsMap[userEntityId] = append(td.userTweetsMap[userEntityId], tw)
		td.usersTweetIdSet[userEntityId][tw.Id] = struct{}{}
		td.count++
	}
	return td.count - oldCount
}

func (td *TweetDumper) Clear() {
	td.userTweetsMap = make(map[int][]*twitterclient.Tweet)
	td.usersTweetIdSet = make(map[int]map[uint64]struct{})
	td.count = 0
}

////////////////////////////////////////////////////////////////////////////////

func (td *TweetDumper) Load(path string) error {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	loaded := make(map[int][]*twitterclient.Tweet)
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	for k, v := range loaded {
		td.Push(k, v...)
	}
	return nil
}

func (td *TweetDumper) Dump(path string) error {
	data, err := json.MarshalIndent(td.userTweetsMap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0666)
}

////////////////////////////////////////////////////////////////////////////////

func (td *TweetDumper) ListAll(ctx context.Context) ([]*dldto.NewEntity, error) {
	res := make([]*dldto.NewEntity, 0, td.count)

	for entityID, userTweets := range td.userTweetsMap {
		userEntity, err := td.userEntityRepo.GetById(ctx, td.db, entityID)
		if err != nil {
			return nil, err
		}
		if userEntity == nil {
			return nil, fmt.Errorf("entity %d is not exists", entityID)
		}

		userSmartPath, err := smartpathdto.NewWithoutDepth(userEntity)
		if err != nil {
			return nil, err
		}

		for _, tw := range userTweets {
			res = append(
				res,
				&dldto.NewEntity{Tweet: tw, Entity: userSmartPath},
			)
		}
	}

	return res, nil
}

func (td *TweetDumper) Count() int {
	return td.count
}

func (td *TweetDumper) GetTweetsByEntityId(userEntityId int) []*twitterclient.Tweet {
	tweets, exists := td.userTweetsMap[userEntityId]
	if !exists {
		return nil
	}

	return tweets
}
