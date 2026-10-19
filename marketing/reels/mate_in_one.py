#!/usr/bin/env python3
"""Composes the "Mate in 1. Easy." reel from a recording of the real app.

The gameplay is recorded, not faked: the iOS app's debug reel studio
(ios/Chess/Reel.swift) plays the scene on the iPhone simulator. This lays the
story over it at 1080 x 1920:

  Hook: "Mate in 1. Easy." over a real Scholar's Mate position.
  The queen is picked up; its lines even show f7.
  Qxh7??, the queen hangs. "bro."
  What every other chess app does: a red, shaking BLUNDER and a buzzer.
  Silence, then the Calm Chess coach: "Oops. Undo and try again."
  Undo. No judgment. "Now find it." Qxf7#: Great move, the king falls.
  The win, and the line that gets it sent to a friend.
  End card, then it fades into its own first frame, so it loops.

Caption, sound and tap timings all follow the beats measured in the recording
(the --pick, --blunder... options), so a re-recording only needs new beats.

Captions sit between Instagram's top bar and its bottom caption area; the
board and the coach card stay clear of both.

Usage: python mate_in_one.py <recording.mov> <out.mp4> [--start 5.0]
Needs Pillow and ffmpeg.
"""
import argparse
import sys
from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter

sys.path.insert(0, str(Path(__file__).resolve().parent))
from reelkit import (BAD, DANGER, GREAT, GREAT_CHIME, INK, MIST, MOVE, SELECT, WIN, W, H, Reel, Track, draw_line, font)


def loud_app(path):
    """What every other chess app does: red, a rating drop, a verdict in capitals."""
    img = Image.new("RGBA", (W, H), (120, 0, 0, 110))
    vignette = Image.new("RGBA", (W, H))
    ImageDraw.Draw(vignette).rectangle((0, 0, W, H), outline=(220, 0, 0, 255), width=170)
    img.alpha_composite(vignette.filter(ImageFilter.GaussianBlur(90)))
    d = ImageDraw.Draw(img)
    label = "every other chess app:"
    f = font(46, 800)
    bw = d.textbbox((0, 0), label, font=f)[2]
    d.rounded_rectangle(((W - bw) / 2 - 28, 296, (W + bw) / 2 + 28, 372), 38, fill=(20, 20, 20, 230))
    d.text(((W - bw) / 2, 305), label, font=f, fill=(255, 255, 255))
    for text, size, y, color, ang in [("?? BLUNDER", 150, 720, (255, 40, 40), -6), ("Rating 1452 → 1428  (−24)", 54, 1010, (255, 255, 255), 3),
                                      ("0:03", 120, 1120, (255, 60, 60), -2), ("You missed mate in 1.", 52, 1290, (255, 255, 255), 0)]:
        layer = Image.new("RGBA", (W, 260))
        lf = font(size, 1000)
        box = ImageDraw.Draw(layer).textbbox((0, 0), text, font=lf)
        tx = (W - (box[2] - box[0])) / 2 - box[0]
        ImageDraw.Draw(layer).text((tx + 6, 26), text, font=lf, fill=(0, 0, 0, 200))
        ImageDraw.Draw(layer).text((tx, 20), text, font=lf, fill=color)
        img.alpha_composite(layer.rotate(ang, resample=Image.BICUBIC), (0, y - 60))
    img.save(path)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("recording")
    ap.add_argument("out")
    # Recording times of each beat, read off the frames (the first frame each shows).
    ap.add_argument("--start", type=float, default=5.3, help="the staged position is on screen")
    ap.add_argument("--pick", type=float, default=7.0, help="the queen is lifted")
    ap.add_argument("--blunder", type=float, default=8.0, help="the queen lands on h7")
    ap.add_argument("--undo", type=float, default=11.0, help="the move is taken back")
    ap.add_argument("--pick2", type=float, default=12.3, help="the queen is lifted again")
    ap.add_argument("--mate", type=float, default=13.3, help="the queen lands on f7")
    ap.add_argument("--won", type=float, default=14.5, help="the finish card arrives")
    ap.add_argument("--stop", type=float, default=16.35, help="the end of the recording")
    a = ap.parse_args()
    freeze_at = a.blunder + 0.4  # the loud app takes over the frame here, for a second
    reel = Reel(a.recording, [("play", a.start, freeze_at), ("freeze", freeze_at, 1.0), ("play", freeze_at, a.stop)])
    loud_app(reel.work / "loud.png")
    pick, blunder, undo, pick2, mate, won = (reel.at(x) for x in (a.pick, a.blunder, a.undo, a.pick2, a.mate, a.won))
    loud = reel.at(freeze_at)  # the frozen second, under the loud app
    calm = loud + 1.0
    reel.overlay(reel.work / "loud.png", loud, calm)
    reel.shake = (loud, calm)
    reel.caption(0.0, 0.6, [("Mate in 1.", INK, 96)])
    reel.caption(0.6, blunder, [("Mate in 1.", INK, 96), ("Easy. 😎", INK, 72)])
    reel.caption(blunder, loud, [("bro. 💀", DANGER, 104)])
    reel.caption(calm, undo, [("Calm Chess just…", INK, 70), ("helps. 🥹", GREAT, 84)])
    reel.caption(undo, pick2, [("Undo.", INK, 88), ("No judgment.", MIST, 62)])
    reel.caption(pick2, mate, [("Now find it. 👀", INK, 84)])
    reel.caption(mate, won, [("CHECKMATE.", GREAT, 104)])
    reel.caption(won, reel.length, [("send this to the friend", INK, 58), ("who hangs their queen 💀", INK, 58)])
    # The finger lands a beat before the board answers.
    for t, sq in [(pick, "h5"), (blunder, "h7"), (pick2, "h5"), (mate, "f7")]:
        reel.ring(t - 0.1, sq)

    total = reel.length + 2.6
    t = Track(total)
    # Tension under the hook: a clock, the thing chess usually rushes you with.
    k = 0
    while 0.15 + k * 0.45 < blunder:
        t.tones(0.15 + k * 0.45, [(2400, 0, 0.05, 0.05), (1200, 0, 0.04, 0.03)])
        k += 1
    t.tones(pick, SELECT, gain=1.6)
    t.tones(blunder, MOVE, gain=1.6)
    t.tones(blunder, [(1980, 0, 0.3, 0.05)], gain=1.4)
    t.noise(blunder, 0.35, 0.12)
    t.buzzer(loud, 1.0)
    t.pad(calm + 0.2, total, 0.035)
    t.tones(calm + 0.25, BAD, triangle=True, gain=1.3)
    t.tones(undo, [(990, 0, 0.1, 0.035)], gain=1.5)
    t.tones(pick2, SELECT, gain=1.6)
    t.tones(mate, MOVE, gain=1.6)
    t.tones(mate + 0.15, GREAT_CHIME, gain=1.5)
    t.tones(won, WIN, gain=1.4)
    t.tones(reel.length + 0.05, [(1046, 0, 0.9, 0.05), (1568, 0.05, 0.9, 0.03)], gain=1.4)
    print(reel.render(a.out, t))


if __name__ == "__main__":
    main()
