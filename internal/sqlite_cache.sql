PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;     -- relax durability a bit for speed
PRAGMA wal_autocheckpoint = 100; -- pages between automatic checkpoints
PRAGMA busy_timeout = 5000;      -- set a longer busy timeout to copy with slow disks

-- Twters table: known authors (feeds/users)
CREATE TABLE IF NOT EXISTS twters (
  uri         TEXT PRIMARY KEY,
  nick        TEXT,
  hashing_uri TEXT,
  metadata    TEXT
);

-- Twts table: individual posts
CREATE TABLE IF NOT EXISTS twts (
  hash           TEXT PRIMARY KEY,
  feed_url       TEXT NOT NULL,
  content        TEXT NOT NULL,
  created        TEXT NOT NULL,
  created_dt     TEXT GENERATED ALWAYS AS (datetime(created)) STORED,
  subject        TEXT DEFAULT '',
  mentions       TEXT DEFAULT '[]',
  tags           TEXT DEFAULT '[]',
  links          TEXT DEFAULT '[]',
  FOREIGN KEY (feed_url) REFERENCES twters(uri) ON DELETE CASCADE
);

-- Feeds table: track known external feeds for fetching
CREATE TABLE IF NOT EXISTS feeds (
  url           TEXT PRIMARY KEY,
  fetches       INTEGER DEFAULT 0,
  errors        INTEGER DEFAULT 0,
  last_error    TEXT DEFAULT '',
  last_fetched  DATETIME,
  last_modified TEXT DEFAULT ''
);

-- Latest Twts table: one entry per feed representing the latest tweet
CREATE TABLE IF NOT EXISTS latest_twts (
  feed_url TEXT PRIMARY KEY,
  hash          TEXT NOT NULL,
  content       TEXT NOT NULL,
  created       TEXT NOT NULL,
  created_dt    TEXT GENERATED ALWAYS AS (datetime(created)) STORED,
  subject       TEXT DEFAULT '',
  mentions      TEXT DEFAULT '[]',
  tags          TEXT DEFAULT '[]',
  links         TEXT DEFAULT '[]',
  FOREIGN KEY (feed_url) REFERENCES twters(uri) ON DELETE CASCADE
);

-- Indexes for performance
CREATE UNIQUE INDEX IF NOT EXISTS idx_feeds_url ON feeds(url);

CREATE INDEX IF NOT EXISTS idx_twts_datetime_created_hash ON twts(datetime(created), hash);
CREATE INDEX IF NOT EXISTS idx_twts_subject_created_hash  ON twts(subject, created, hash);
CREATE INDEX IF NOT EXISTS idx_twts_feed_created ON twts(feed_url, created);
CREATE INDEX IF NOT EXISTS idx_twts_hash            ON twts(hash);

CREATE INDEX IF NOT EXISTS idx_twters_uri           ON twters(uri);
CREATE INDEX IF NOT EXISTS idx_twters_lower_nick    ON twters(LOWER(nick));
CREATE INDEX IF NOT EXISTS idx_twters_lower_uri     ON twters(LOWER(uri));

-- Index for ordering or filtering by the computed timestamp
CREATE INDEX IF NOT EXISTS idx_twts_created_dt ON twts(created_dt);

-- Composite index to speed up queries filtering by feed and ordering by created_dt
CREATE INDEX IF NOT EXISTS idx_twts_feed_created_dt ON twts(feed_url, created_dt);

CREATE INDEX IF NOT EXISTS idx_twts_feed_created_dt_hash 
  ON twts(feed_url, created_dt, hash, subject);

-- FTS5 virtual table for full-text search
CREATE VIRTUAL TABLE IF NOT EXISTS twts_fts USING fts5(
    hash UNINDEXED,
    content,
    created UNINDEXED,
    feed_url UNINDEXED,
    nick UNINDEXED,
    metadata UNINDEXED,
    tokenize = 'porter'
);

-- Trigger to update the FTS5 table after an insert
CREATE TRIGGER IF NOT EXISTS twts_ai AFTER INSERT ON twts
BEGIN
  INSERT INTO twts_fts (hash, content, created, feed_url, nick, metadata)
  SELECT
    new.hash,
    new.content,
    new.created,
    new.feed_url,
    tw.nick,
    tw.metadata
  FROM twters tw
  WHERE tw.uri = new.feed_url;
END;

-- Trigger to update the FTS5 table after a delete
CREATE TRIGGER IF NOT EXISTS twts_ad AFTER DELETE ON twts
BEGIN
  DELETE FROM twts_fts WHERE hash = old.hash;
END;

-- Trigger to update the latest_twts table after an insert on the twts table.
CREATE TRIGGER IF NOT EXISTS trg_update_latest_twts_after_insert
AFTER INSERT ON twts
FOR EACH ROW
BEGIN
  -- Insert the tweet into latest_twts if no row exists for this feed
  INSERT OR IGNORE INTO latest_twts (feed_url, hash, content, created, subject, mentions, tags, links)
  VALUES (NEW.feed_url, NEW.hash, NEW.content, NEW.created, NEW.subject, NEW.mentions, NEW.tags, NEW.links);

  -- If the new tweet is more recent than the stored one, update the record
  UPDATE latest_twts
  SET hash = NEW.hash,
      content = NEW.content,
      created = NEW.created,
      subject = NEW.subject,
      mentions = NEW.mentions,
      tags = NEW.tags,
      links = NEW.links
  WHERE feed_url = NEW.feed_url
    AND NEW.created > created;
END;