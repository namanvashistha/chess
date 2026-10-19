"""Shared parts of the Calm Chess reels: captions in SF Rounded, the app's
synthesized sounds, tap marks, the end card, and a timeline that cuts and
freezes a simulator recording of the real app and lays everything over it at
1080 x 1920.

A reel is a Reel: its segments of the recording, captions, overlays, tap
marks and a soundtrack, then render().
"""
import math
import random
import shutil
import struct
import subprocess
import tempfile
import wave
from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter, ImageFont

W, H = 1080, 1920
FPS = 30
ROOT = Path(__file__).resolve().parents[2]
ICON = ROOT / "ios/Chess/Assets.xcassets/AppIcon.appiconset/AppIcon.png"
ROUNDED = "/System/Library/Fonts/SFNSRounded.ttf"
EMOJI = "/System/Library/Fonts/Apple Color Emoji.ttc"

# The app's light palette.
INK = (34, 39, 44)
MIST = (104, 112, 118)
ACCENT = (46, 97, 80)
GREAT = (47, 158, 92)
DANGER = (194, 59, 59)
BG_TOP = (241, 239, 233)
BG_BOTTOM = (230, 227, 220)

# Where the recording sits: the board and coach card, cropped from the
# simulator frame (1206 x 2622) and scaled to 900 wide.
CROP_Y, CROP_H, APP_W, APP_X, APP_Y = 400, 1540, 900, 90, 490
K = APP_W / 1206


def square(name):
    """A square's centre on the reel, from where the board sits in the recording."""
    f = ord(name[0]) - 97
    r = 8 - int(name[1])
    x, y = 43 + (f + 0.5) * 140, 434 + (r + 0.5) * 139.6
    return APP_X + x * K, APP_Y + (y - CROP_Y) * K


def font(size, weight=900):
    f = ImageFont.truetype(ROUNDED, size)
    f.set_variation_by_axes([400, weight])
    return f


def is_emoji(ch):
    """Drawn from Apple Color Emoji: pictographs, and clocks and the like from 0x2300 up."""
    return ord(ch) >= 0x2300 and ch not in "−→·…♚"


def draw_line(img, text, y, size, color, weight=900, center=True, x=None):
    """One line of text with colour emoji, centred (or from x) at y. Returns its height."""
    d = ImageDraw.Draw(img)
    f = font(size, weight)
    runs, cur = [], ""
    for ch in text:
        if is_emoji(ch):
            if cur:
                runs.append(("t", cur))
                cur = ""
            runs.append(("e", ch))
        elif ch != "️":
            cur += ch
    if cur:
        runs.append(("t", cur))
    pieces = []
    for kind, s in runs:
        if kind == "t":
            box = d.textbbox((0, 0), s, font=f)
            pieces.append((kind, s, box[2] - box[0], box))
        else:
            e = Image.new("RGBA", (180, 180))
            ImageDraw.Draw(e).text((0, 0), s, font=ImageFont.truetype(EMOJI, 160), embedded_color=True)
            e = e.crop(e.getbbox()).resize((int(size * 1.05), int(size * 1.05)), Image.LANCZOS)
            pieces.append((kind, e, e.width, None))
    total = sum(p[2] for p in pieces) + 10 * (len(pieces) - 1)
    cx = (W - total) / 2 if center else x
    for kind, s, w, box in pieces:
        if kind == "t":
            d.text((cx - box[0], y), s, font=f, fill=color)
        else:
            img.alpha_composite(s, (int(cx), int(y + size * 0.08)))
        cx += w + 10
    return size * 1.2


def caption(lines, path):
    """Captions in the band above the board, on a soft card so they read over anything."""
    img = Image.new("RGBA", (W, H))
    heights = sum(l[2] * 1.2 for l in lines)
    top = 300 + (180 - heights) / 2
    card = Image.new("RGBA", (W, H))
    ImageDraw.Draw(card).rounded_rectangle((70, top - 26, W - 70, top + heights + 18), 34, fill=(251, 250, 247, 235))
    shadow = card.filter(ImageFilter.GaussianBlur(18))
    shadow = Image.eval(shadow, lambda v: v)
    img.alpha_composite(Image.composite(Image.new("RGBA", (W, H), (0, 0, 0, 40)), Image.new("RGBA", (W, H)), shadow.split()[3]), (0, 10))
    img.alpha_composite(card)
    y = top
    for text, color, size in lines:
        y += draw_line(img, text, y, size, color)
    img.save(path)


def background(path):
    img = Image.new("RGB", (W, H))
    d = ImageDraw.Draw(img)
    for y in range(H):
        t = y / H
        d.line([(0, y), (W, y)], fill=tuple(int(a + (b - a) * t) for a, b in zip(BG_TOP, BG_BOTTOM)))
    glow = Image.new("RGBA", (W, H))
    ImageDraw.Draw(glow).ellipse((W / 2 - 520, -520, W / 2 + 520, 520), fill=(255, 250, 240, 160))
    img = img.convert("RGBA")
    img.alpha_composite(glow.filter(ImageFilter.GaussianBlur(160)))
    img.convert("RGB").save(path)


def tap_ring(path):
    """Where the finger lands: a soft ring of light."""
    s = 150
    img = Image.new("RGBA", (s, s))
    d = ImageDraw.Draw(img)
    d.ellipse((20, 20, s - 20, s - 20), fill=(255, 255, 255, 90), outline=(255, 255, 255, 235), width=7)
    img.filter(ImageFilter.GaussianBlur(0.6)).save(path)


def end_card(path):
    """The promise, the name, and why it's different. The same on every reel, so tests compare formats, not endings."""
    img = Image.open(path.parent / "bg.png").convert("RGBA")
    icon = Image.open(ICON).convert("RGBA").resize((300, 300), Image.LANCZOS)
    mask = Image.new("L", (300, 300))
    ImageDraw.Draw(mask).rounded_rectangle((0, 0, 299, 299), 68, fill=255)
    shadow = Image.new("RGBA", (W, H))
    ImageDraw.Draw(shadow).rounded_rectangle((390, 470, 690, 770), 68, fill=(0, 0, 0, 70))
    img.alpha_composite(shadow.filter(ImageFilter.GaussianBlur(26)))
    img.paste(icon, (390, 450), mask)
    y = 820
    y += draw_line(img, "Chess that doesn't", y, 84, INK)
    y += draw_line(img, "yell at you.", y, 84, INK)
    y += 40
    y += draw_line(img, "Calm Chess", y, 64, ACCENT)
    y += draw_line(img, "Free · No ads · Works offline", y + 10, 40, MIST, 600)
    draw_line(img, "iPhone & Android", y + 30, 40, MIST, 600)
    img.convert("RGB").save(path)


# Sound: the app's own synthesized tones, plus a clock and the loud app's buzzer.
RATE = 44100


class Track:
    def __init__(self, seconds):
        self.s = [0.0] * int(seconds * RATE)

    def tones(self, at, partials, square=False, triangle=False, gain=1.0):
        for freq, start, dur, g in partials:
            s0 = int((at + start) * RATE)
            for k in range(int(dur * RATE)):
                i = s0 + k
                if i >= len(self.s):
                    break
                t = k / RATE
                ph = 2 * math.pi * freq * t
                w = math.sin(ph)
                if square:
                    w = 1.0 if w >= 0 else -1.0
                if triangle:
                    w = 2 / math.pi * math.asin(math.sin(ph))
                self.s[i] += w * g * gain * min(1, t / 0.005) * math.exp(-5 * t / dur)

    def noise(self, at, dur, gain):
        prev = 0.0
        n = int(dur * RATE)
        for k in range(n):
            i = int(at * RATE) + k
            if i >= len(self.s):
                break
            white = random.uniform(-1, 1)
            self.s[i] += (white - prev) * gain * (1 - k / n) ** 3 * 0.5
            prev = white

    def buzzer(self, at, dur):
        """A harsh, detuned buzz under a low boom: the sound of being scolded."""
        for k in range(int(dur * RATE)):
            i = int(at * RATE) + k
            if i >= len(self.s):
                break
            t = k / RATE
            env = min(1, t / 0.01) * (1 - t / dur) ** 0.6
            buzz = sum((1 if math.sin(2 * math.pi * f * t) >= 0 else -1) for f in (146.8, 155.6, 293.7)) / 3
            boom = math.sin(2 * math.pi * (60 - 25 * t) * t) * math.exp(-4 * t)
            self.s[i] += 0.16 * buzz * env + 0.5 * boom

    def pad(self, start, end, gain):
        """A soft, slow chord: the calm after."""
        notes = [220.0, 277.18, 329.63, 415.30]
        for i in range(int(start * RATE), min(len(self.s), int(end * RATE))):
            t = i / RATE - start
            span = end - start
            env = min(1, t / 1.2) * min(1, (span - t) / 1.0)
            v = sum(math.sin(2 * math.pi * f * t + n) for n, f in enumerate(notes)) / len(notes)
            self.s[i] += v * gain * env * (0.85 + 0.15 * math.sin(2 * math.pi * 0.25 * t))

    def write(self, path, fade_out_at):
        peak = max(abs(v) for v in self.s) or 1
        scale = 0.89 / peak
        with wave.open(str(path), "wb") as w:
            w.setnchannels(2)
            w.setsampwidth(2)
            w.setframerate(RATE)
            frames = bytearray()
            for i, v in enumerate(self.s):
                t = i / RATE
                g = 1.0 if t < fade_out_at else max(0.0, 1 - (t - fade_out_at) / 0.4)
                x = int(max(-1, min(1, v * scale * g)) * 32767)
                frames += struct.pack("<hh", x, x)
            w.writeframes(bytes(frames))


SELECT = [(1320, 0, 0.12, 0.05)]
MOVE = [(880, 0, 0.25, 0.09), (1760, 0, 0.18, 0.03)]
GREAT_CHIME = [(784, 0, 0.35, 0.07), (988, 0.07, 0.35, 0.07), (1319, 0.14, 0.35, 0.07)]
BAD = [(330, 0, 0.3, 0.07), (311, 0.08, 0.35, 0.05)]
WIN = [(523, 0, 0.5, 0.07), (659, 0.09, 0.5, 0.07), (784, 0.18, 0.5, 0.07), (1046, 0.27, 0.5, 0.07), (1319, 0.36, 0.5, 0.07)]


def run(cmd):
    subprocess.run(cmd, check=True)




def countdown_bar(folder, seconds):
    """A bar that drains over `seconds`, as frames: folder/bar_000.png onwards."""
    n = int(seconds * FPS)
    for i in range(n + 1):
        img = Image.new("RGBA", (W, 40))
        d = ImageDraw.Draw(img)
        d.rounded_rectangle((110, 8, W - 110, 32), 12, fill=(34, 39, 44, 40))
        left = 1 - i / n
        if left > 0:
            color = DANGER if left < 0.34 else ACCENT
            d.rounded_rectangle((110, 8, 110 + max(24, (W - 220) * left), 32), 12, fill=color + (255,))
        img.save(folder / f"bar_{i:03d}.png")


class Reel:
    """A recording of the app, cut into segments, with everything laid over it.

    segments: ("play", from, to) plays that stretch of the recording;
    ("freeze", at, seconds) holds the frame at `at`.
    """

    def __init__(self, recording, segments):
        self.recording = recording
        self.segments = segments
        self.work = Path(tempfile.mkdtemp(prefix="reel-"))
        background(self.work / "bg.png")
        tap_ring(self.work / "ring.png")
        end_card(self.work / "end.png")
        # Lengths in whole frames, as the cuts are made.
        self.length = round(sum(round(((b - a) if kind == "play" else b) * FPS) for kind, a, b in segments) / FPS, 3)
        self.captions, self.overlays, self.rings, self.sequences = [], [], [], []
        self.shake = None

    def at(self, raw):
        """Reel time of a moment in the recording (it must be in a played segment)."""
        frames = 0
        for kind, a, b in self.segments:
            if kind == "play":
                if a <= raw <= b:
                    return round((frames + round(raw * FPS) - round(a * FPS)) / FPS, 3)
                frames += round((b - a) * FPS)
            else:
                frames += round(b * FPS)
        raise ValueError(f"{raw} is cut from the reel")

    def caption(self, t0, t1, lines):
        path = self.work / f"cap{len(self.captions)}.png"
        caption(lines, path)
        self.captions.append((path, t0, t1))

    def overlay(self, path, t0, t1):
        self.overlays.append((path, t0, t1))

    def ring(self, t, sq):
        self.rings.append((t, sq))

    def sequence(self, pattern, t0, t1):
        self.sequences.append((pattern, t0, t1))

    def render(self, out, track, end_len=2.6):
        main_len, total = self.length, self.length + end_len
        track.write(self.work / "audio.wav", total - 0.45)
        # The simulator records at a variable frame rate; cutting that drifts by a
        # frame or two per cut. A constant-rate copy cuts on exact frames.
        steady = self.work / "steady.mp4"
        run(["ffmpeg", "-loglevel", "error", "-y", "-i", str(self.recording), "-vf", f"fps={FPS}", "-an",
             "-c:v", "libx264", "-crf", "10", "-preset", "veryfast", str(steady)])
        app = f"crop=1206:{CROP_H}:0:{CROP_Y},scale={APP_W}:-2"
        n = len(self.segments)
        fc = [f"[0:v]split={n}" + "".join(f"[s{i}]" for i in range(n))]
        for i, (kind, a, b) in enumerate(self.segments):
            first = round(a * FPS)
            if kind == "play":
                fc.append(f"[s{i}]trim=start_frame={first}:end_frame={first + round((b - a) * FPS)},setpts=PTS-STARTPTS,{app}[g{i}]")
            else:
                fc.append(f"[s{i}]trim=start_frame={first}:end_frame={first + 1},setpts=PTS-STARTPTS,{app},"
                          f"loop=loop={round(b * FPS) - 1}:size=1,setpts=N/{FPS}/TB[g{i}]")
        fc.append("".join(f"[g{i}]" for i in range(n)) + f"concat=n={n}:v=1:a=0[game]")
        fc.append(f"[1:v]loop=loop=-1:size=1,trim=0:{main_len},fps={FPS},setpts=PTS-STARTPTS[bg]")
        fc.append(f"[bg][game]overlay={APP_X}:{APP_Y}:shortest=1[base]")
        inputs, k, last = [], 2, "base"
        for path, t0, t1 in self.overlays:
            inputs += ["-loop", "1", "-t", str(main_len), "-i", str(path)]
            fc.append(f"[{last}][{k}:v]overlay=0:0:enable='between(t,{t0},{t1})'[o{k}]")
            last, k = f"o{k}", k + 1
        if self.shake:
            lo, hi = self.shake
            fc.append(f"[{last}]scale=1134:2016,crop=1080:1920:'27+if(between(t,{lo},{hi}),18*sin(t*97),0)':"
                      f"'48+if(between(t,{lo},{hi}),14*cos(t*83),0)'[shaken]")
            last = "shaken"
        for path, t0, t1 in self.captions:
            inputs += ["-loop", "1", "-t", str(main_len), "-i", str(path)]
            fc.append(f"[{last}][{k}:v]overlay=0:0:enable='between(t,{t0},{t1})'[o{k}]")
            last, k = f"o{k}", k + 1
        for pattern, t0, t1 in self.sequences:
            inputs += ["-framerate", str(FPS), "-i", str(pattern)]
            fc.append(f"[{k}:v]setpts=PTS+{t0}/TB[q{k}]")
            fc.append(f"[{last}][q{k}]overlay=0:478:eof_action=pass:enable='between(t,{t0},{t1})'[o{k}]")
            last, k = f"o{k}", k + 1
        for t, sq in self.rings:
            x, y = square(sq)
            inputs += ["-loop", "1", "-t", str(main_len), "-i", str(self.work / "ring.png")]
            fc.append(f"[{k}:v]format=rgba,fade=t=out:st={t + 0.05}:d=0.35:alpha=1[ring{k}]")
            fc.append(f"[{last}][ring{k}]overlay={x - 75:.0f}:{y - 75:.0f}:enable='between(t,{t - 0.05},{t + 0.4})'[o{k}]")
            last, k = f"o{k}", k + 1
        # The end card fades in over the last frame, then the reel's own first frame
        # fades in over the card, so it loops without a seam.
        inputs += ["-loop", "1", "-t", str(total), "-i", str(self.work / "end.png")]
        fc.append(f"[{last}]tpad=stop_mode=clone:stop_duration={end_len},split[main][first]")
        fc.append(f"[{k}:v]fps={FPS},format=rgba,fade=t=in:st={main_len - 0.3}:d=0.3:alpha=1[end]")
        fc.append(f"[first]trim=0:0.04,setpts=PTS-STARTPTS,loop=loop={int(total * FPS)}:size=1,setpts=N/{FPS}/TB,format=rgba,"
                  f"fade=t=in:st={total - 0.45}:d=0.4:alpha=1[still]")
        fc.append(f"[main][end]overlay=0:0:enable='gte(t,{main_len - 0.3})'[withend]")
        fc.append(f"[withend][still]overlay=0:0:enable='gte(t,{total - 0.45})',format=yuv420p[v]")
        k += 1
        run(["ffmpeg", "-loglevel", "error", "-y", "-i", str(steady), "-loop", "1", "-i", str(self.work / "bg.png"),
             *inputs, "-i", str(self.work / "audio.wav"), "-filter_complex", ";".join(fc), "-map", "[v]", "-map", f"{k}:a",
             "-c:v", "libx264", "-preset", "slow", "-crf", "17", "-pix_fmt", "yuv420p", "-r", str(FPS),
             "-af", "loudnorm=I=-14:TP=-1.5:LRA=11", "-c:a", "aac", "-b:a", "192k", "-ar", "44100", "-shortest",
             "-movflags", "+faststart", str(out)])
        # The reel's first frame is its cover.
        run(["ffmpeg", "-loglevel", "error", "-y", "-ss", "0.5", "-i", str(out), "-frames:v", "1",
             str(Path(out).with_suffix(".cover.jpg"))])
        return out
