-- Initial schema for PostgreSQL
CREATE TABLE IF NOT EXISTS users (
	id BIGINT NOT NULL, 
	screen_name VARCHAR NOT NULL, 
	name VARCHAR NOT NULL, 
	protected BOOLEAN NOT NULL, 
	friends_count INTEGER NOT NULL, 
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id), 
	UNIQUE (screen_name)
);

CREATE TABLE IF NOT EXISTS user_previous_names (
	id SERIAL PRIMARY KEY, 
	uid BIGINT NOT NULL, 
	screen_name VARCHAR NOT NULL, 
	name VARCHAR NOT NULL, 
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
	FOREIGN KEY(uid) REFERENCES users (id)
);

CREATE TABLE IF NOT EXISTS lsts (
	id BIGINT NOT NULL, 
	name VARCHAR NOT NULL, 
	owner_uid BIGINT NOT NULL, 
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS lst_entities (
	id SERIAL PRIMARY KEY, 
	lst_id BIGINT NOT NULL, 
	name VARCHAR NOT NULL, 
	parent_dir VARCHAR NOT NULL, 
	folder_name VARCHAR NOT NULL,
	storage_saved BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	UNIQUE (lst_id, parent_dir)
);

CREATE TABLE IF NOT EXISTS user_entities (
	id SERIAL PRIMARY KEY,
	user_id BIGINT NOT NULL UNIQUE, 
	name VARCHAR NOT NULL, 
	parent_dir VARCHAR NOT NULL, 
	folder_name VARCHAR NOT NULL,
	storage_saved BOOLEAN NOT NULL DEFAULT FALSE,
	media_count INTEGER,
	latest_release_time TIMESTAMP, 
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	UNIQUE (user_id, parent_dir), 
	FOREIGN KEY(user_id) REFERENCES users (id)
);

CREATE TABLE IF NOT EXISTS user_links (
	id SERIAL PRIMARY KEY,
	user_id BIGINT NOT NULL, 
	name VARCHAR NOT NULL, 
	parent_lst_entity_id INTEGER NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	UNIQUE (user_id, parent_lst_entity_id),
	FOREIGN KEY(user_id) REFERENCES users (id), 
	FOREIGN KEY(parent_lst_entity_id) REFERENCES lst_entities (id)
);

CREATE TABLE IF NOT EXISTS tweets (
	id SERIAL PRIMARY KEY,
	user_id BIGINT NOT NULL,
	tweet_id BIGINT NOT NULL,
	content TEXT NOT NULL,
	tweet_time TIMESTAMP NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(user_id) REFERENCES users (id),
	UNIQUE(tweet_id)
);

CREATE TABLE IF NOT EXISTS medias (
	id SERIAL PRIMARY KEY,
	user_id BIGINT NOT NULL,
	tweet_id BIGINT NOT NULL,
	location VARCHAR NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(user_id) REFERENCES users (id),
	FOREIGN KEY(tweet_id) REFERENCES tweets (id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_user_links_user_id ON user_links (user_id);
CREATE INDEX IF NOT EXISTS idx_tweets_user_id ON tweets (user_id);
CREATE INDEX IF NOT EXISTS idx_tweets_tweet_id ON tweets (tweet_id);
CREATE INDEX IF NOT EXISTS idx_medias_user_id ON medias (user_id);
CREATE INDEX IF NOT EXISTS idx_medias_tweet_id ON medias (tweet_id);
CREATE INDEX IF NOT EXISTS idx_tweets_tweet_time ON tweets (tweet_time);
