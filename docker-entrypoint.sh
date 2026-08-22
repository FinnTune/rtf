#!/bin/bash -e
if [ ! -f database/forum.db ]; then
  echo "No database found at database/forum.db - creating one from database/createTables.sql..."
  (cd database && sqlite3 forum.db < createTables.sql)
fi
exec "$@"
