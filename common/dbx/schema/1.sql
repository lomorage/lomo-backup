CREATE TABLE IF NOT EXISTS dirs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  path VARCHAR NOT NULL,
  scan_root_dir_id INTEGER NOT NULL,
  mod_time TIMESTAMP,
  create_time TIMESTAMP NOT NULL,

  UNIQUE(scan_root_dir_id, path)
);

CREATE TABLE IF NOT EXISTS files (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  dir_id INTEGER NOT NULL,
  iso_id INTEGER NOT NULL DEFAULT 0,
  name VARCHAR NOT NULL,
  ext VARCHAR NOT NULL,
  size INTEGER NOT NULL,
  hash VARCHAR NOT NULL,
  mod_time TIMESTAMP NOT NULL,
  create_time TIMESTAMP NOT NULL,

  UNIQUE(dir_id, name)
);

CREATE TABLE IF NOT EXISTS isos (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name VARCHAR NOT NULL UNIQUE,
  region VARCHAR DEFAULT "" NOT NULL,
  bucket VARCHAR DEFAULT "" NOT NULL,
  size INTEGER NOT NULL,
  status INTEGER NOT NULL,
  hash_hex VARCHAR NOT NULL,
  hash_base64 VARCHAR DEFAULT "" NOT NULL,
  upload_key VARCHAR DEFAULT "" NOT NULL,
  upload_id VARCHAR DEFAULT "" NOT NULL,
  create_time TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS parts (
  iso_id INTEGER NOT NULL,
  part_no INTEGER NOT NULL,
  hash_hex VARCHAR NOT NULL,
  hash_base64 VARCHAR NOT NULL,
  etag VARCHAR DEFAULT "" NOT NULL,
  size INTEGER,
  status INTEGER,
  create_time TIMESTAMP NOT NULL,

  CONSTRAINT iso_part UNIQUE (iso_id, part_no)
);

