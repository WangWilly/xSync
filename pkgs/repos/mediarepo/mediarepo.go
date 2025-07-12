package mediarepo

import (
	"database/sql"
	"time"

	"github.com/WangWilly/xSync/pkgs/model"
	"github.com/jmoiron/sqlx"
)

type Repo struct{}

func New() *Repo {
	return &Repo{}
}

func (r *Repo) Create(db *sqlx.DB, media *model.Media) error {
	stmt := `INSERT INTO medias(user_id, tweet_id, location) 
			 VALUES(:user_id, :tweet_id, :location)`
	res, err := db.NamedExec(stmt, media)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	media.Id = id
	return nil
}

func (r *Repo) GetById(db *sqlx.DB, id int64) (*model.Media, error) {
	stmt := `SELECT * FROM medias WHERE id=?`
	result := &model.Media{}
	err := db.Get(result, stmt, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return result, err
}

func (r *Repo) GetByUserId(db *sqlx.DB, userId uint64) ([]*model.Media, error) {
	stmt := `SELECT * FROM medias WHERE user_id=? ORDER BY created_at DESC`
	var medias []*model.Media
	err := db.Select(&medias, stmt, userId)
	return medias, err
}

func (r *Repo) GetByTweetId(db *sqlx.DB, tweetId int64) ([]*model.Media, error) {
	stmt := `SELECT * FROM medias WHERE tweet_id=? ORDER BY created_at ASC`
	var medias []*model.Media
	err := db.Select(&medias, stmt, tweetId)
	return medias, err
}

func (r *Repo) GetByLocation(db *sqlx.DB, location string) (*model.Media, error) {
	stmt := `SELECT * FROM medias WHERE location=?`
	result := &model.Media{}
	err := db.Get(result, stmt, location)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return result, err
}

func (r *Repo) Update(db *sqlx.DB, media *model.Media) error {
	media.UpdatedAt = time.Now()
	stmt := `UPDATE medias SET tweet_id=?, location=?, updated_at=? WHERE id=?`
	_, err := db.Exec(stmt, media.TweetId, media.Location, media.UpdatedAt, media.Id)
	return err
}

func (r *Repo) Delete(db *sqlx.DB, id int64) error {
	stmt := `DELETE FROM medias WHERE id=?`
	_, err := db.Exec(stmt, id)
	return err
}
