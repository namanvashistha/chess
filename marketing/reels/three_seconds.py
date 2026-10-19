#!/usr/bin/env python3
"""Composes the "Only 3% see it in 3 seconds" reel from a recording of the real app.

The A/B partner of "Mate in 1. Easy.": where that one shows someone failing
and lets viewers correct them, this one challenges the viewer directly. Same
length, same end card and the same sounds, so a test compares the formats.

The position is the Petrov trap (1.e4 e5 2.Nf3 Nf6 3.Nxe5 Nxe4 4.Qe2 Nf6??):
Nc6+ is a discovered check from the queen, and the knight attacks the black
queen, so it wins her. The debug reel studio (ios/Chess/Reel.swift, scene
"three-seconds") plays it on the iPhone simulator: the countdown is held, the
app's own two-step hint finds the knight and then the move, Nc6+, the bishop
blocks, and the knight takes the queen.

Dead time is cut: the hint's reading pause and Black's thinking.

Usage: python three_seconds.py <recording.mov> <out.mp4> [beat options]
"""
import argparse
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from reelkit import (DANGER, GREAT, GREAT_CHIME, INK, MOVE, SELECT, Reel, Track, countdown_bar)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("recording")
    ap.add_argument("out")
    # Recording times of each beat, read off the frames (the first frame each shows).
    ap.add_argument("--start", type=float, default=5.6, help="the staged position is on screen")
    ap.add_argument("--hint", type=float, default=9.6, help="the hint circles the knight")
    ap.add_argument("--hint2", type=float, default=11.65, help="the hint shows the move")
    ap.add_argument("--pick", type=float, default=14.1, help="the knight is lifted")
    ap.add_argument("--move", type=float, default=14.85, help="the knight lands on c6, with check")
    ap.add_argument("--reply", type=float, default=17.8, help="the bishop blocks on e7")
    ap.add_argument("--pick2", type=float, default=18.8, help="the knight is lifted again")
    ap.add_argument("--take", type=float, default=19.5, help="the knight takes the queen")
    ap.add_argument("--stop", type=float, default=20.35, help="the end of the recording")
    a = ap.parse_args()

    clock = 3.0  # the viewer's three seconds
    reel = Reel(a.recording, [
        ("play", a.start, a.start + clock + 0.6),       # the challenge, then time's up
        ("play", a.hint - 0.15, a.hint2 + 1.2),          # the hint, both steps
        ("play", a.pick - 0.2, a.move + 0.8),            # the move
        ("play", a.reply - 0.25, a.stop),                # the block, and the queen goes
        ("freeze", a.stop, 1.4),                         # held, for the line to send
    ])
    hint, hint2, pick, move, reply, pick2, take = (reel.at(x) for x in (a.hint, a.hint2, a.pick, a.move, a.reply, a.pick2, a.take))

    countdown_bar(reel.work, clock)
    reel.sequence(reel.work / "bar_%03d.png", 0.0, clock)
    reel.caption(0.0, clock, [("Only 3% see it", INK, 92), ("in 3 seconds. 👀", INK, 74)])
    reel.caption(clock, hint, [("Time's up. ⏰", DANGER, 100)])
    reel.caption(hint, hint2, [("Need a hint? 🤫", INK, 88)])
    reel.caption(hint2, move, [("Knight to c6…", INK, 88)])
    reel.caption(move, reply, [("DISCOVERED", GREAT, 92), ("CHECK. 🤯", GREAT, 92)])
    reel.caption(reply, take, [("Block it?", INK, 88), ("Too late.", DANGER, 72)])
    reel.caption(take, take + 1.3, [("Queen: gone. 👑", GREAT, 92)])
    reel.caption(take + 1.3, reel.length, [("send this to someone", INK, 60), ("who'd never see it 😏", INK, 60)])
    for t, sq in [(pick, "e5"), (move, "c6"), (pick2, "c6"), (take, "d8")]:
        reel.ring(t - 0.1, sq)

    total = reel.length + 2.6
    t = Track(total)
    # The viewer's clock: a firm tick each second, softer ones between.
    for k in range(int(clock * 4)):
        firm = k % 4 == 0
        t.tones(0.05 + k * 0.25, [(2400, 0, 0.05, 0.06 if firm else 0.025), (1200, 0, 0.04, 0.035 if firm else 0.015)])
    t.tones(clock, [(880, 0, 0.25, 0.06), (880, 0.18, 0.3, 0.06)], square=True, gain=0.8)  # time's up
    t.pad(hint, total, 0.035)
    t.tones(hint, SELECT, gain=1.6)
    t.tones(hint2, [(1046, 0, 0.25, 0.06)], gain=1.6)
    t.tones(pick, SELECT, gain=1.6)
    t.tones(move, MOVE, gain=1.6)
    t.tones(move + 0.05, [(660, 0, 0.2, 0.04), (660, 0.14, 0.2, 0.04)], square=True, gain=1.2)  # check
    t.tones(move + 0.25, GREAT_CHIME, gain=1.5)
    t.tones(reply, MOVE, gain=1.3)
    t.tones(pick2, SELECT, gain=1.6)
    t.tones(take, MOVE, gain=1.6)
    t.tones(take, [(1980, 0, 0.3, 0.05)], gain=1.4)
    t.noise(take, 0.35, 0.12)
    t.tones(take + 0.2, GREAT_CHIME, gain=1.6)
    t.tones(reel.length + 0.05, [(1046, 0, 0.9, 0.05), (1568, 0.05, 0.9, 0.03)], gain=1.4)
    print(reel.render(a.out, t))


if __name__ == "__main__":
    main()
