#!/usr/bin/env bash
# Checks that all apps answer the same status, headers and body on the 4 routes.
# Date and the value of X-Request-Id differ by design and are masked.
# The first app of APPS is the reference. Exit 1 on any difference.
set -euo pipefail
cd "$(dirname "$0")/.."
. scripts/lib.sh

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"; app_down' EXIT

snapshot() {
  for path in /plaintext /json /users/42 /mw/json; do
    echo "### GET $path"
    curl -si --http1.1 "$BASE_URL$path" | tr -d '\r' | awk '
      BEGIN { body = 0 }
      body { print "BODY " $0; next }
      /^$/ { body = 1; next }
      NR == 1 { print "STATUS " $2; next }
      {
        name = tolower(substr($0, 1, index($0, ":") - 1))
        value = substr($0, index($0, ":") + 2)
        if (name == "date") { print "date: <set>"; next }
        if (name == "x-request-id") { print "x-request-id: " (value == "" ? "<EMPTY>" : "<set>"); next }
        print name ": " value
      }' | sort
  done
}

for app in $APPS; do
  app_down
  app_up "$app"
  snapshot >"$tmp/$app.txt"
done

ref="${APPS%% *}"
status=0
for app in $APPS; do
  [ "$app" = "$ref" ] && continue
  if diff -u "$tmp/$ref.txt" "$tmp/$app.txt" >"$tmp/diff.txt"; then
    echo "verify: $app == $ref"
  else
    echo "verify: $app DIFFERS from $ref"
    cat "$tmp/diff.txt"
    status=1
  fi
done
echo "--- reference ($ref)"
cat "$tmp/$ref.txt"
exit $status
