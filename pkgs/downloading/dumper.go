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
	userEntityRepo UserEntityRepo

	data  map[int][]*twitterclient.Tweet
	set   map[int]map[uint64]struct{}
	count int
}

func NewDumper() *TweetDumper {
	td := TweetDumper{
		userEntityRepo: userentityrepo.New(),
	}
	td.data = make(map[int][]*twitterclient.Tweet)
	td.set = make(map[int]map[uint64]struct{})
	return &td
}

func (td *TweetDumper) Push(eid int, tweet ...*twitterclient.Tweet) int {
	_, ok := td.data[eid]
	if !ok {
		td.data[eid] = make([]*twitterclient.Tweet, 0, len(tweet))
		td.set[eid] = make(map[uint64]struct{})
	}

	oldCount := td.count

	for _, tw := range tweet {
		_, exist := td.set[eid][tw.Id]
		if exist {
			continue
		}
		td.data[eid] = append(td.data[eid], tw)
		td.set[eid][tw.Id] = struct{}{}
		td.count++
	}
	return td.count - oldCount
}

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
	err = json.Unmarshal(data, &loaded)
	if err != nil {
		return err
	}

	for k, v := range loaded {
		td.Push(k, v...)
	}
	return nil
}

func (td *TweetDumper) Dump(path string) error {
	data, err := json.MarshalIndent(td.data, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0666)
}

func (td *TweetDumper) Clear() {
	td.data = make(map[int][]*twitterclient.Tweet)
	td.set = make(map[int]map[uint64]struct{})
	td.count = 0
}

func (td *TweetDumper) GetTotal(db *sqlx.DB) ([]*dldto.NewEntity, error) {
	ctx := context.Background()

	results := make([]*dldto.NewEntity, 0, td.count)

	for k, v := range td.data {
		e, err := td.userEntityRepo.GetById(ctx, db, k)
		if err != nil {
			return nil, err
		}
		if e == nil {
			return nil, fmt.Errorf("entity %d is not exists", k)
		}
		ue, err := smartpathdto.RebuildUserSmartPath(e)
		if err != nil {
			return nil, err
		}

		for _, tw := range v {
			results = append(results, &dldto.NewEntity{Tweet: tw, Entity: ue})
		}
	}
	return results, nil
}

func (td *TweetDumper) Count() int {
	return td.count
}

// GetTweetsByEntityId returns tweets for a specific entity ID
func (td *TweetDumper) GetTweetsByEntityId(entityId int) []*twitterclient.Tweet {
	tweets, exists := td.data[entityId]
	if !exists {
		return nil
	}
	return tweets
}

// GetAllEntities returns all entity IDs that have tweets
func (td *TweetDumper) GetAllEntities() []int {
	var entities []int
	for entityId := range td.data {
		entities = append(entities, entityId)
	}
	return entities
}
