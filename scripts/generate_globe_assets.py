"""Generate the RYZE globe assets from Natural Earth land polygons (public domain).

Outputs:
  - an equirectangular land mask (2048x1024, 16 grey levels, lossless WebP) used
    by the desktop 3D Earth in components/home_story/story_scene.ts;
  - components/brand_globe/globe_data.ts: the route cities, the routes, and a
    pre-projected orthographic SVG globe for phones.

Inputs: ne_50m_land.geojson and ne_110m_land.geojson from
https://github.com/nvkelso/natural-earth-vector/tree/master/geojson
(Natural Earth, public domain). Requires Pillow. Run from the repository root:

  python3 -I scripts/generate_globe_assets.py <natural_earth_dir> \
      main/src/resources/img/globe/earth_land_mask.webp \
      main/src/frontend/src/components/brand_globe/globe_data.ts
"""
import json, math, sys
from PIL import Image, ImageDraw

ne_dir, mask_out, ts_out = sys.argv[1:4]

def rings(path):
    data = json.load(open(path))
    for f in data["features"]:
        g = f["geometry"]
        polys = g["coordinates"] if g["type"] == "MultiPolygon" else [g["coordinates"]]
        for poly in polys:
            yield poly  # [exterior, holes...]

# ---------------- equirectangular land mask ----------------
W, H = 4096, 2048
img = Image.new("L", (W, H), 0)
d = ImageDraw.Draw(img)
xy = lambda lon, lat: ((lon + 180) / 360 * W, (90 - lat) / 180 * H)
for poly in rings(f"{ne_dir}/ne_50m_land.geojson"):
    d.polygon([xy(*p) for p in poly[0]], fill=255)
    for hole in poly[1:]:
        d.polygon([xy(*p) for p in hole], fill=0)
mask = img.resize((2048, 1024), Image.LANCZOS)
# 16 grey levels keep the anti-aliased coastline while the lossless file stays small.
mask.point(lambda v: round(v / 17) * 17).save(mask_out, "WEBP", lossless=True, method=6)

# ---------------- routes ----------------
CITIES = {
    "lisbon": (38.72, -9.14), "london": (51.51, -0.13), "paris": (48.86, 2.35), "berlin": (52.52, 13.40),
    "new_york": (40.71, -74.01), "toronto": (43.65, -79.38), "mexico_city": (19.43, -99.13),
    "los_angeles": (34.05, -118.24), "sao_paulo": (-23.55, -46.63), "luanda": (-8.84, 13.23),
    "lagos": (6.52, 3.38), "cape_town": (-33.92, 18.42), "maputo": (-25.97, 32.57), "dubai": (25.20, 55.27),
    "mumbai": (19.08, 72.88), "singapore": (1.35, 103.82), "tokyo": (35.68, 139.69), "sydney": (-33.87, 151.21),
}
HOME = "lisbon"
ROUTES = [("lisbon", c) for c in ["london", "berlin", "new_york", "toronto", "sao_paulo", "luanda", "lagos", "maputo", "cape_town", "dubai"]] + [
    ("new_york", "los_angeles"), ("toronto", "mexico_city"), ("dubai", "mumbai"), ("dubai", "singapore"),
    ("singapore", "sydney"), ("singapore", "tokyo"), ("los_angeles", "tokyo"), ("london", "paris")]

def vec(lat, lon):
    la, lo = math.radians(lat), math.radians(lon)
    return (math.cos(la) * math.cos(lo), math.cos(la) * math.sin(lo), math.sin(la))

def slerp(a, b, t):
    dot = max(-1, min(1, sum(x * y for x, y in zip(a, b))))
    om = math.acos(dot)
    if om < 1e-6:
        return a
    s = math.sin(om)
    return tuple((math.sin((1 - t) * om) * x + math.sin(t * om) * y) / s for x, y in zip(a, b))

# ---------------- orthographic SVG globe (mobile) ----------------
LAT0, LON0, R = 24.0, -16.0, 100.0
la0, lo0 = math.radians(LAT0), math.radians(LON0)
def ortho(lat, lon, alt=0.0):
    la, lo = math.radians(lat), math.radians(lon)
    k = R * (1 + alt)
    x = k * math.cos(la) * math.sin(lo - lo0)
    y = -k * (math.cos(la0) * math.sin(la) - math.sin(la0) * math.cos(la) * math.cos(lo - lo0))
    z = math.sin(la0) * math.sin(la) + math.cos(la0) * math.cos(la) * math.cos(lo - lo0)
    return x, y, z

def project_ring(ring):
    pts = []
    for lon, lat in ring:
        x, y, z = ortho(lat, lon)
        if z < 0:  # behind: push onto the horizon along its azimuth
            r = math.hypot(x, y) or 1
            x, y = x / r * R, y / r * R
        pts.append((x, y))
    return pts

def rdp(pts, eps):
    if len(pts) < 3:
        return pts
    (x1, y1), (x2, y2) = pts[0], pts[-1]
    dx, dy = x2 - x1, y2 - y1
    n = math.hypot(dx, dy) or 1e-9
    best, idx = -1, 0
    for i in range(1, len(pts) - 1):
        px, py = pts[i]
        dist = abs(dy * px - dx * py + x2 * y1 - y2 * x1) / n
        if dist > best:
            best, idx = dist, i
    if best > eps:
        return rdp(pts[: idx + 1], eps)[:-1] + rdp(pts[idx:], eps)
    return [pts[0], pts[-1]]

def area(pts):
    return abs(sum(pts[i][0] * pts[i - 1][1] - pts[i - 1][0] * pts[i][1] for i in range(len(pts)))) / 2

parts = []
for poly in rings(f"{ne_dir}/ne_110m_land.geojson"):
    ring = poly[0]
    if all(ortho(lat, lon)[2] < 0 for lon, lat in ring):
        continue
    proj = project_ring(ring)
    if proj[0] == proj[-1]:
        proj = proj[:-1]
    far = max(range(len(proj)), key=lambda i: (proj[i][0] - proj[0][0]) ** 2 + (proj[i][1] - proj[0][1]) ** 2)
    pts = rdp(proj[: far + 1], 0.35)[:-1] + rdp(proj[far:] + [proj[0]], 0.35)[:-1]
    if len(pts) < 3 or area(pts) < 1.5:
        continue
    parts.append("M" + "L".join(f"{x:.1f} {y:.1f}" for x, y in pts) + "Z")
land_path = "".join(parts)

arc_paths, dots = [], []
for a, b in ROUTES:
    pa, pb = CITIES[a], CITIES[b]
    if ortho(*pa)[2] < 0.12 or ortho(*pb)[2] < 0.12:
        continue
    va, vb = vec(*pa), vec(*pb)
    ang = math.acos(max(-1, min(1, sum(x * y for x, y in zip(va, vb)))))
    lift = 0.06 + 0.22 * ang / math.pi
    pts = []
    for i in range(33):
        t = i / 32
        v = slerp(va, vb, t)
        lat = math.degrees(math.asin(v[2])); lon = math.degrees(math.atan2(v[1], v[0]))
        x, y, _ = ortho(lat, lon, lift * math.sin(math.pi * t))
        pts.append((x, y))
    pts = rdp(pts, 0.15)
    arc_paths.append({"from": a, "to": b, "d": "M" + "L".join(f"{x:.1f} {y:.1f}" for x, y in pts)})
for name, (lat, lon) in CITIES.items():
    x, y, z = ortho(lat, lon)
    if z > 0.12 and any(name in r for r in ROUTES):
        dots.append({"id": name, "x": round(x, 1), "y": round(y, 1)})

ts = [
    "// Generated from Natural Earth land polygons (public domain, naturalearthdata.com).",
    "// Generated by scripts/generate_globe_assets.py; do not edit by hand.",
    "",
    "export interface GlobeCity { id: string; lat: number; lon: number }",
    "",
    f"export const GLOBE_HOME = \"{HOME}\";",
    "",
    "export const GLOBE_CITIES: GlobeCity[] = [",
    *[f"  {{ id: \"{k}\", lat: {v[0]}, lon: {v[1]} }}," for k, v in CITIES.items()],
    "];",
    "",
    "export const GLOBE_ROUTES: Array<[string, string]> = [",
    *[f"  [\"{a}\", \"{b}\"]," for a, b in ROUTES],
    "];",
    "",
    f"// Orthographic view centred on {LAT0}N {abs(LON0)}W, radius {int(R)} (viewBox -100 -100 200 200).",
    f"export const GLOBE_SVG_LAND = \"{land_path}\";",
    "",
    "export const GLOBE_SVG_ARCS: Array<{ from: string; to: string; d: string }> = [",
    *[f"  {{ from: \"{x['from']}\", to: \"{x['to']}\", d: \"{x['d']}\" }}," for x in arc_paths],
    "];",
    "",
    "export const GLOBE_SVG_CITIES: Array<{ id: string; x: number; y: number }> = [",
    *[f"  {{ id: \"{x['id']}\", x: {x['x']}, y: {x['y']} }}," for x in dots],
    "];",
    "",
]
open(ts_out, "w").write("\n".join(ts))
print("land path chars", len(land_path), "arcs", len(arc_paths), "dots", len(dots))
