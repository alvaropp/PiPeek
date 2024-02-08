#!/bin/bash

# Load environment variables
set -a
. /etc/monitor/.env
set +a

# Replace placeholders in index.html
envsubst '$MONGO_CLUSTER_NAME,$MONGO_DATABASE,$MONGO_MONITOR_COLLECTIONS,$MONGO_APP_ID,$MONGO_MONITOR_KEY' < /usr/share/nginx/html/index.html > /usr/share/nginx/html/index.html.tmp
mv /usr/share/nginx/html/index.html.tmp /usr/share/nginx/html/index.html

# Start Nginx
exec "$@"
