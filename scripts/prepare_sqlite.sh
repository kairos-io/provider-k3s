#!/bin/bash

set -e

db_file="/etc/kubernetes/state.sqlite3"

# SQL commands to create the 'kine' table
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
PRAGMA wal_checkpoint(TRUNCATE);
"

# Execute the SQL commands using sqlite3
echo "$sql_commands" | sqlite3 "$db_file"

# Print a success message
echo "Table 'kine' created successfully"
