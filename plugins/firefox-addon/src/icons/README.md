# Firefox Addon Icons

Required PNGs (checked into this directory):

- `icon-16.png` — toolbar / favicon
- `icon-48.png` — toolbar
- `icon-96.png` — about / management page

These are referenced by `manifest.json`. Regenerate with ImageMagick if needed:

```bash
cd plugins/firefox-addon/src/icons
for size in 16 48 96; do
  pad=$((size/8))
  magick -size ${size}x${size} xc:none \
    -fill '#0F766E' -draw "roundrectangle ${pad},${pad} $((size-pad-1)),$((size-pad-1)) $((size/5)),$((size/5))" \
    -fill '#99F6E4' -draw "circle $((size/2)),$((size/2)) $((size/2)),$((size/2 - size/5))" \
    -fill '#0F766E' -draw "circle $((size/2)),$((size/2)) $((size/2)),$((size/2 - size/10))" \
    "icon-${size}.png"
done
```
