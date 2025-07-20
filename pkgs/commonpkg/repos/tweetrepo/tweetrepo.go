package tweetrepo

import (
	"database/sql"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

type Repo struct{}

func New() *Repo {
	return &Repo{}
}

func (r *Repo) Create(db *sqlx.DB, tweet *model.Tweet) error {
	stmt := `INSERT INTO tweets(user_id, tweet_id, content, tweet_time) 
			VALUES(:user_id, :tweet_id, :content, :tweet_time)
			RETURNING id, user_id, tweet_id, content, tweet_time, created_at, updated_at`
	rows, err := db.NamedQuery(stmt, tweet)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return sql.ErrNoRows
	}
	if err := rows.StructScan(tweet); err != nil {
		return err
	}
	return nil
}

func (r *Repo) GetById(db *sqlx.DB, id int64) (*model.Tweet, error) {
	stmt := `SELECT * FROM tweets WHERE id=$1`
	result := &model.Tweet{}
	err := db.Get(result, stmt, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return result, err
}

func (r *Repo) GetByUserId(db *sqlx.DB, userId uint64) ([]*model.Tweet, error) {
	stmt := `SELECT * FROM tweets WHERE user_id=$1 ORDER BY tweet_time DESC`
	var tweets []*model.Tweet
	err := db.Select(&tweets, stmt, userId)
	return tweets, err
}

func (r *Repo) Update(db *sqlx.DB, tweet *model.Tweet) error {
	tweet.UpdatedAt = time.Now()
	stmt := `UPDATE tweets SET tweet_id=$1, content=$2, tweet_time=$3, updated_at=$4 WHERE id=$5`
	_, err := db.Exec(stmt, tweet.TweetId, tweet.Content, tweet.TweetTime, tweet.UpdatedAt, tweet.Id)
	return err
}

func (r *Repo) Delete(db *sqlx.DB, id int64) error {
	stmt := `DELETE FROM tweets WHERE id=$1`
	_, err := db.Exec(stmt, id)
	return err
}

func (r *Repo) GetWithMedia(db *sqlx.DB, userId uint64) ([]map[string]interface{}, error) {
	stmt := `SELECT t.*, m.location as media_location 
			 FROM tweets t 
			 LEFT JOIN medias m ON t.id = m.tweet_id 
			 WHERE t.user_id=$1 
			 ORDER BY t.tweet_time DESC`

	rows, err := db.Query(stmt, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var tweet model.Tweet
		var mediaLocation sql.NullString

		err := rows.Scan(&tweet.Id, &tweet.UserId, &tweet.TweetId, &tweet.Content,
			&tweet.TweetTime, &tweet.CreatedAt, &tweet.UpdatedAt, &mediaLocation)
		if err != nil {
			return nil, err
		}

		result := map[string]interface{}{
			"id":         tweet.Id,
			"tweet_id":   tweet.TweetId,
			"content":    tweet.Content,
			"tweet_time": tweet.TweetTime,
			"created_at": tweet.CreatedAt,
			"updated_at": tweet.UpdatedAt,
		}

		if mediaLocation.Valid {
			result["media_location"] = mediaLocation.String
		}

		results = append(results, result)
	}

	return results, nil
}

func (r *Repo) GetByTweetId(db *sqlx.DB, tweetId uint64) (*model.Tweet, error) {
	stmt := `SELECT * FROM tweets WHERE tweet_id=$1`
	result := &model.Tweet{}
	err := db.Get(result, stmt, tweetId)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return result, err
}
