#!/bin/bash
if [[ -z $1 ]]; then
	echo "Syntax: $0 <filename> [lang]" >&2
	exit 1
fi
lang=_auto
if [[ ! -z $2 ]]; then
	lang=$2
fi
oi=$IFS
IFS='|'
read code url < <(curl -fs -w '%{http_code}|%{redirect_url}' --data-urlencode text@"$1" --data-urlencode lang="$lang" http://ghostbin.com/paste/new | sed -e 's/HTTP/http/g')
IFS="$oi"
if [[ $code -ne 200 && $code -ne 303 && $code -ne 302 ]]; then
	echo "Rejected: $code" >&2
	exit 1
fi
echo "$url"
echo -n "$url" | pbcopy