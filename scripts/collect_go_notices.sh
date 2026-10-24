#!/bin/sh
set -eu
mkdir -p /notices/go
cp "$(go env GOROOT)/LICENSE" /notices/go/LICENSE
[ ! -f "$(go env GOROOT)/PATENTS" ] || cp "$(go env GOROOT)/PATENTS" /notices/go/PATENTS
go list -deps -f '{{with .Module}}{{.Path}} {{.Version}} {{.Dir}}{{end}}' ./cmd/snitch | sort -u | while read -r module version directory; do
  [ -n "$directory" ] || continue
  target="/notices/go/$module@$version"
  mkdir -p "$target"
  found=0
  for file in "$directory"/LICENSE* "$directory"/NOTICE* "$directory"/COPYRIGHT* "$directory"/PATENTS*; do
    [ -f "$file" ] || continue
    cp "$file" "$target/"
    found=1
  done
  [ "$found" -eq 1 ] || { echo "Missing license: $module" >&2; exit 1; }
done
