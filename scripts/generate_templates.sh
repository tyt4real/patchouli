#!/usr/bin/env sh
set -euo pipefail

if ! command -v templ >/dev/null 2>&1; then
  echo "templ CLI not found. Install with: go install github.com/a-h/templ/cmd/templ@latest" >&2
  exit 1
fi

for f in web/templ/*.templ; do
  [ -e "$f" ] || continue
  name=$(basename "$f" .templ)
  out="web/views/${name}_templ_gen.go"
  echo "generating ${out} from ${f}"
  templ "$f" > "$out"
done

echo "templates generated into web/views"
