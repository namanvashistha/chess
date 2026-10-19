#!/bin/sh
# Builds every reel from reels.py, renders it with Remotion, and masters its
# sound to -14 LUFS (where Reels sit), into ../out/<id>.mp4 with a cover frame.
# Uses the Apple-silicon Node, so Remotion renders with a native Chrome.
set -e
cd "$(dirname "$0")"
export PATH=/opt/homebrew/bin:$PATH
uv run -q --with chess reels.py
mkdir -p out ../out
for id in ${@:-opera en-passant stalemate knights-tour fools-mate}; do
  npx remotion render "$id" "out/$id.mp4" --concurrency=6 --log=error
  ffmpeg -loglevel error -y -i "out/$id.mp4" -c:v copy -af "loudnorm=I=-14:TP=-1.5:LRA=11" -c:a aac -b:a 192k -ar 44100 \
    -movflags +faststart "../out/$id.mp4"
  ffmpeg -loglevel error -y -ss 0.2 -i "../out/$id.mp4" -frames:v 1 "../out/$id.cover.jpg"
  echo "../out/$id.mp4"
done
