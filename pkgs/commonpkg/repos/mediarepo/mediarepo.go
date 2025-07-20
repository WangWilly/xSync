package mediarepo

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

type Repo struct{}

func New() *Repo {
	return &Repo{}
}

////////////////////////////////////////////////////////////////////////////////

func (r *Repo) Create(db *sqlx.DB, media *model.Media) error {
	stmt := `INSERT INTO medias(user_id, tweet_id, location) 
			VALUES(:user_id, :tweet_id, :location)
			RETURNING id, user_id, tweet_id, location, created_at, updated_at`
	rows, err := db.NamedQuery(stmt, media)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no rows returned for media with user_id %d and tweet_id %d", media.UserId, media.TweetId)
	}
	if err := rows.StructScan(media); err != nil {
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (r *Repo) GetById(db *sqlx.DB, id int64) (*model.Media, error) {
	stmt := `SELECT * FROM medias WHERE id=$1`
	result := &model.Media{}
	err := db.Get(result, stmt, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return result, err
}

func (r *Repo) GetByUserId(db *sqlx.DB, userId uint64) ([]*model.Media, error) {
	stmt := `SELECT * FROM medias WHERE user_id=$1 ORDER BY created_at DESC`
	var medias []*model.Media
	err := db.Select(&medias, stmt, userId)
	return medias, err
}

func (r *Repo) GetByTweetId(db *sqlx.DB, tweetId int64) ([]*model.Media, error) {
	stmt := `SELECT * FROM medias WHERE tweet_id=$1 ORDER BY created_at ASC`
	var medias []*model.Media
	err := db.Select(&medias, stmt, tweetId)
	return medias, err
}

func (r *Repo) GetByLocation(db *sqlx.DB, location string) (*model.Media, error) {
	stmt := `SELECT * FROM medias WHERE location=$1`
	result := &model.Media{}
	err := db.Get(result, stmt, location)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return result, err
}

////////////////////////////////////////////////////////////////////////////////

func (r *Repo) Update(db *sqlx.DB, media *model.Media) error {
	media.UpdatedAt = time.Now()
	stmt := `UPDATE medias SET tweet_id=$1, location=$2, updated_at=$3 WHERE id=$4 RETURNING updated_at`

	var updatedAt time.Time
	err := db.QueryRow(stmt, media.TweetId, media.Location, media.UpdatedAt, media.Id).Scan(&updatedAt)
	if err != nil {
		return err
	}

	// Update the struct with the actual database timestamp
	media.UpdatedAt = updatedAt
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (r *Repo) Delete(db *sqlx.DB, id int64) error {
	stmt := `DELETE FROM medias WHERE id=$1`
	_, err := db.Exec(stmt, id)
	return err
}
