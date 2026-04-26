#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${1:-http://localhost:3001}"
RESPONSE_FILE="$(mktemp)"
SUFFIX="$(date +%H%M%S)"

cleanup() {
  rm -f "$RESPONSE_FILE"
}
trap cleanup EXIT

json_get() {
  ruby -rjson -e '
    data = JSON.parse(File.read(ARGV[0]))
    ARGV[1].split(".").each { |key| data = data.fetch(key) }
    puts data
  ' "$RESPONSE_FILE" "$1"
}

request() {
  method="$1"
  path="$2"
  token="$3"
  expected="$4"
  body="${5:-}"

  curl_args=(-sS -o "$RESPONSE_FILE" -w "%{http_code}" -X "$method")
  if [ -n "$token" ]; then
    curl_args+=(-H "Authorization: Bearer $token")
  fi

  if [ -n "$body" ]; then
    curl_args+=(-H "Content-Type: application/json" -d "$body")
    status=$(curl "${curl_args[@]}" "$BASE_URL$path")
  else
    status=$(curl "${curl_args[@]}" "$BASE_URL$path")
  fi

  if [ "$status" != "$expected" ]; then
    echo "FAIL $method $path expected HTTP $expected but got HTTP $status"
    cat "$RESPONSE_FILE"
    echo
    exit 1
  fi

  echo "OK   $method $path -> HTTP $status"
}

photo="$(printf 'wasatext-smoke-photo' | base64 | tr -d '\n')"

medi="Medi$SUFFIX"
lina="Lina$SUFFIX"
nuri="Nuri$SUFFIX"
group="G$SUFFIX"
renamed_group="Team$SUFFIX"
renamed_medi="Med$SUFFIX"

echo "Running WASAText API smoke test against $BASE_URL"

request POST /session "" 201 "{\"name\":\"$medi\"}"
medi_id="$(json_get identifier)"

request POST /session "" 201 "{\"name\":\"$lina\"}"
lina_id="$(json_get identifier)"

request POST /session "" 201 "{\"name\":\"$nuri\"}"
nuri_id="$(json_get identifier)"

request PUT /users/me/name "$medi_id" 200 "{\"name\":\"$renamed_medi\"}"
request PUT /users/me/photo "$medi_id" 200 "{\"photo\":\"$photo\"}"
request GET "/users?name=$lina" "$medi_id" 200
request GET /conversations "$medi_id" 200

request POST /conversations "$medi_id" 201 "{\"userId\":\"$lina_id\"}"
private_conversation_id="$(json_get id)"

request GET "/conversations/$private_conversation_id" "$medi_id" 200

request POST "/conversations/$private_conversation_id/messages" "$medi_id" 201 "{\"type\":\"text\",\"text\":\"hello from smoke test\"}"
message_id="$(json_get id)"

request POST "/conversations/$private_conversation_id/messages" "$medi_id" 201 "{\"type\":\"photo\",\"photo\":\"$photo\"}"

request POST "/conversations/$private_conversation_id/messages" "$lina_id" 201 "{\"type\":\"text\",\"text\":\"reply from Lina\",\"replyToMessageId\":\"$message_id\"}"
reply_id="$(json_get id)"

request PUT "/messages/$message_id/comments/me" "$lina_id" 200 "{\"emoticon\":\":ok:\"}"
request DELETE "/messages/$message_id/comments/me" "$lina_id" 204

request POST /conversations "$medi_id" 201 "{\"userId\":\"$nuri_id\"}"
destination_conversation_id="$(json_get id)"

request POST "/messages/$message_id/forward" "$medi_id" 201 "{\"conversationId\":\"$destination_conversation_id\"}"
request DELETE "/messages/$reply_id" "$lina_id" 204

request POST /groups "$medi_id" 201 "{\"name\":\"$group\",\"memberIds\":[\"$lina_id\"]}"
group_id="$(json_get id)"

request PUT "/groups/$group_id/name" "$medi_id" 200 "{\"name\":\"$renamed_group\"}"
request PUT "/groups/$group_id/photo" "$medi_id" 200 "{\"photo\":\"$photo\"}"
request PUT "/groups/$group_id/members/$nuri_id" "$medi_id" 200
request DELETE "/groups/$group_id/members/me" "$medi_id" 204

echo "Smoke test passed."
