#!/bin/bash

set -e

db_dir="/var/lib/rancher/k3s/server/db"

mkdir -p $db_dir

# SQL commands to create the 'kine' and 'ssh_keys' tables
sql_commands="
CREATE TABLE IF NOT EXISTS kine (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name INTEGER,
	created INTEGER,
	deleted INTEGER,
	create_revision INTEGER,
	prev_revision INTEGER,
	lease INTEGER,
	value BLOB,
	old_value BLOB
);
CREATE INDEX IF NOT EXISTS kine_name_index ON kine (name);
CREATE INDEX IF NOT EXISTS kine_name_id_index ON kine (name,id);
CREATE INDEX IF NOT EXISTS kine_id_deleted_index ON kine (id,deleted);
CREATE INDEX IF NOT EXISTS kine_prev_revision_index ON kine (prev_revision);
CREATE UNIQUE INDEX IF NOT EXISTS kine_name_prev_revision_uindex ON kine (name, prev_revision);

CREATE TABLE IF NOT EXISTS ssh_keys (
	id INTEGER NOT NULL PRIMARY KEY,
	host TEXT NOT NULL,
	public_key TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS ssh_keys_host_index ON ssh_keys (host);
CREATE INDEX IF NOT EXISTS ssh_keys_host_id_index ON ssh_keys (host,id);

PRAGMA wal_checkpoint(TRUNCATE);
"

# Execute the SQL commands using sqlite3
echo "$sql_commands" | sqlite3 "$db_dir/state.db"

# Print a success message
echo "sqlite database initialized successfully"
