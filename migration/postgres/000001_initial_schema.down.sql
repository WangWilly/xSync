-- Drop tables in reverse order due to foreign key constraints
DROP INDEX IF EXISTS idx_tweets_tweet_time;
DROP INDEX IF EXISTS idx_medias_tweet_id;
DROP INDEX IF EXISTS idx_medias_user_id;
DROP INDEX IF EXISTS idx_tweets_tweet_id;
DROP INDEX IF EXISTS idx_tweets_user_id;
DROP INDEX IF EXISTS idx_user_links_user_id;

DROP TABLE IF EXISTS medias;
DROP TABLE IF EXISTS tweets;
DROP TABLE IF EXISTS user_links;
DROP TABLE IF EXISTS user_entities;
DROP TABLE IF EXISTS lst_entities;
DROP TABLE IF EXISTS user_previous_names;
DROP TABLE IF EXISTS lsts;
DROP TABLE IF EXISTS users;
