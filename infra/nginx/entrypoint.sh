#!/bin/sh
set -e

# Substitute only our own variables so nginx built-in variables like
# $host, $remote_addr, $http_upgrade etc. are left untouched.
envsubst '$DOMAIN $PORT_LANDING $PORT_ADMIN $PORT_SUPPORT $PORT_BACKEND' \
  < /etc/nginx/templates/nginx.conf.template \
  > /etc/nginx/conf.d/default.conf

exec nginx -g 'daemon off;'
