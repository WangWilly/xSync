package model

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const Schema = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER NOT NULL, 
	screen_name VARCHAR NOT NULL, 
	name VARCHAR NOT NULL, 
	protected BOOLEAN NOT NULL, 
	friends_count INTEGER NOT NULL, 
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id), 
	UNIQUE (screen_name)
);

CREATE TABLE IF NOT EXISTS user_previous_names (
	id INTEGER NOT NULL, 
	uid INTEGER NOT NULL, 
	screen_name VARCHAR NOT NULL, 
	name VARCHAR NOT NULL, 
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP, 
	PRIMARY KEY (id), 
	FOREIGN KEY(uid) REFERENCES users (id)
);

CREATE TABLE IF NOT EXISTS lsts (
	id INTEGER NOT NULL, 
	name VARCHAR NOT NULL, 
	owner_uid INTEGER NOT NULL, 
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS lst_entities (
	id INTEGER NOT NULL, 
	lst_id INTEGER NOT NULL, 
	name VARCHAR NOT NULL, 
	parent_dir VARCHAR NOT NULL COLLATE NOCASE, 
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id), 
	UNIQUE (lst_id, parent_dir)
);

CREATE TABLE IF NOT EXISTS user_entities (
	id INTEGER NOT NULL, 
	user_id INTEGER NOT NULL, 
	name VARCHAR NOT NULL, 
	latest_release_time DATETIME, 
	parent_dir VARCHAR COLLATE NOCASE NOT NULL, 
	media_count INTEGER,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id), 
	UNIQUE (user_id, parent_dir), 
	FOREIGN KEY(user_id) REFERENCES users (id)
);

CREATE TABLE IF NOT EXISTS user_links (
	id INTEGER NOT NULL,
	user_id INTEGER NOT NULL, 
	name VARCHAR NOT NULL, 
	parent_lst_entity_id INTEGER NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	UNIQUE (user_id, parent_lst_entity_id),
	FOREIGN KEY(user_id) REFERENCES users (id), 
	FOREIGN KEY(parent_lst_entity_id) REFERENCES lst_entities (id)
);

CREATE INDEX IF NOT EXISTS idx_user_links_user_id ON user_links (user_id);

CREATE TABLE IF NOT EXISTS tweets (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	tweet_id INTEGER NOT NULL,
	content TEXT NOT NULL,
	tweet_time DATETIME NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(user_id) REFERENCES users (id),
	UNIQUE(tweet_id)
);

CREATE TABLE IF NOT EXISTS medias (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	tweet_id INTEGER NOT NULL,
	location VARCHAR NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(user_id) REFERENCES users (id),
	FOREIGN KEY(tweet_id) REFERENCES tweets (id)
);

CREATE INDEX IF NOT EXISTS idx_tweets_user_id ON tweets (user_id);
CREATE INDEX IF NOT EXISTS idx_tweets_tweet_id ON tweets (tweet_id);
CREATE INDEX IF NOT EXISTS idx_medias_user_id ON medias (user_id);
CREATE INDEX IF NOT EXISTS idx_medias_tweet_id ON medias (tweet_id);
CREATE INDEX IF NOT EXISTS idx_tweets_tweet_time ON tweets (tweet_time);
`

func CreateTables(db *sqlx.DB) {
	db.MustExec(Schema)
}
