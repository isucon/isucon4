#!/bin/bash

if test "$SLACK_URL" = ""
then
    echo "please set \$SLACK_URL"
    exit 1
fi
exec tee >(while read line; do curl -s --data-binary '`'"$line"'`' "$SLACK_URL" -o/dev/null; done)
