#!/usr/bin/env python3
"""Kitty graphics probe for rdxt figures.

Draws a generated test image two ways inside a figure frame:
  A. direct placement (the image sits at the cursor)
  B. Unicode placeholders (the image is tied to text cells and scrolls with them)
and records the terminal's replies and cell size.

Usage: image-probe.py RESULTS.json
"""
import base64, json, math, os, re, select, struct, sys, termios, time, tty, zlib

OUT = sys.argv[1] if len(sys.argv) > 1 else "image-probe.json"
E = "\x1b"
fd = sys.stdin.fileno()
res = {}


def w(s):
    sys.stdout.write(s)
    sys.stdout.flush()


def read_until(pat, timeout=1.0):
    buf, end = b"", time.time() + timeout
    while time.time() < end:
        r, _, _ = select.select([fd], [], [], max(0, end - time.time()))
        if not r:
            break
        buf += os.read(fd, 65536)
        if re.search(pat, buf):
            break
    return buf.decode("utf-8", "replace")


def query(seq):
    w(seq + E + "[c")
    return read_until(rb"\x1b\[\?[\d;]*c")


def png(width, height):
    """A test image: a soft gradient with a circle and a frame, as PNG."""
    rows = []
    cx, cy, rad = width * 0.62, height * 0.5, height * 0.3
    for y in range(height):
        row = bytearray([0])
        for x in range(width):
            t = x / width
            r, g, b = int(40 + 150 * t), int(60 + 80 * (y / height)), int(200 - 120 * t)
            if (x - cx) ** 2 + (y - cy) ** 2 < rad ** 2:
                r, g, b = 250, 200, 80
            if x < 6 or y < 6 or x >= width - 6 or y >= height - 6:
                r, g, b = 255, 255, 255
            if abs(y - height * 0.25) < 3 and width * 0.1 < x < width * 0.4:
                r, g, b = 255, 255, 255
            row += bytes((r, g, b))
        rows.append(bytes(row))
    raw = zlib.compress(b"".join(rows), 9)

    def chunk(tag, data):
        c = struct.pack(">I", len(data)) + tag + data
        return c + struct.pack(">I", zlib.crc32(tag + data) & 0xFFFFFFFF)

    return b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", width, height, 8, 2, 0, 0, 0)) + \
        chunk(b"IDAT", raw) + chunk(b"IEND", b"")


def transmit(data, keys):
    """Send a PNG with the given control keys, in 4096-byte chunks."""
    b64 = base64.b64encode(data).decode()
    parts = [b64[i:i + 4096] for i in range(0, len(b64), 4096)]
    for i, p in enumerate(parts):
        more = 1 if i < len(parts) - 1 else 0
        ctrl = (keys + f",m={more}") if i == 0 else f"m={more}"
        w(f"{E}_G{ctrl};{p}{E}\\")


# Row and column diacritics for Unicode placeholders (start of kitty's list).
DIACRITICS = [0x0305, 0x030D, 0x030E, 0x0310, 0x0312, 0x033D, 0x033E, 0x033F, 0x0346, 0x034A,
              0x034B, 0x034C, 0x0350, 0x0351, 0x0352, 0x0357, 0x035B, 0x0363, 0x0364, 0x0365]
PLACEHOLDER = "\U0010EEEE"

old = termios.tcgetattr(fd)
try:
    tty.setraw(fd)
    w(E + "[2J" + E + "[H")
    cols_term = os.get_terminal_size().columns
    resp = query(E + "[16t")
    m = re.search(r"\x1b\[6;(\d+);(\d+)t", resp)
    cell_h, cell_w = (int(m.group(1)), int(m.group(2))) if m else (16, 8)
    res["cell_px"] = {"w": cell_w, "h": cell_h}

    img_w, img_h = 640, 360
    data = png(img_w, img_h)
    res["png_bytes"] = len(data)
    fig_cols = min(56, cols_term - 10)
    fig_rows = max(3, round(fig_cols * cell_w * img_h / img_w / cell_h))
    fig_rows = min(fig_rows, 12)
    res["figure_cells"] = {"cols": fig_cols, "rows": fig_rows}
    margin = (cols_term - fig_cols - 4) // 2
    pad = " " * margin
    dim, it, rst = E + "[90m", E + "[3m", E + "[0m"

    def frame_top(label):
        w(f"{pad}{dim}╭{'─' * (fig_cols + 2)}╮{rst}   {E}[1m{label}{rst}\r\n")

    def frame_bottom(caption):
        w(f"{pad}{dim}╰{'─' * (fig_cols + 2)}╯{rst}\r\n")
        w(f"{pad}  {dim}{it}{caption}{rst}\r\n\r\n")

    w(f"\r\n{pad}{E}[1mrdxt image probe{rst}{dim}   cell {cell_w}×{cell_h}px · figure {fig_cols}×{fig_rows} cells{rst}\r\n\r\n")

    # A. Direct placement.
    frame_top("A · direct placement")
    start = time.time()
    for r in range(fig_rows):
        w(f"{pad}{dim}│{rst} " + " " * fig_cols + f" {dim}│{rst}\r\n")
    # Go back up and place the image at the first row inside the frame.
    w(f"{E}[{fig_rows}A{E}[{margin + 3}G")
    transmit(data, f"a=T,f=100,i=41,c={fig_cols},r={fig_rows},C=1,q=0")
    res["direct_reply"] = read_until(rb"\x1b\\", 1.0)
    res["direct_ms"] = round((time.time() - start) * 1000)
    w(f"{E}[{fig_rows}B\r")
    frame_bottom("Figure 1 · A generated test image, placed directly")

    # B. Unicode placeholders: transmit once as a virtual placement, then
    # draw placeholder characters colored with the image's id.
    frame_top("B · Unicode placeholders")
    transmit(data, f"a=T,f=100,i=42,U=1,c={fig_cols},r={fig_rows},q=0")
    res["placeholder_reply"] = read_until(rb"\x1b\\", 1.0)
    for r in range(fig_rows):
        cells = PLACEHOLDER + chr(DIACRITICS[r]) + chr(DIACRITICS[0]) + PLACEHOLDER * (fig_cols - 1)
        w(f"{pad}{dim}│{rst} {E}[38;5;42m{cells}{rst} {dim}│{rst}\r\n")
    frame_bottom("Figure 2 · The same image, tied to text cells")

    w(f"{pad}{dim}Does A show the image? Does B? Press Enter to finish.{rst}\r\n")
    with open(OUT, "w") as f:
        json.dump(res, f, indent=2, default=str)
    while True:
        r, _, _ = select.select([fd], [], [], 600)
        if not r or os.read(fd, 64):
            break
finally:
    w(E + "_Ga=d,d=A,q=2" + E + "\\")
    termios.tcsetattr(fd, termios.TCSADRAIN, old)
