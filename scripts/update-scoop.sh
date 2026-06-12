#!/usr/bin/env bash
# Regenera y publica el manifiesto de Scoop (omusic.json) en el bucket
# (por defecto AlexCas/scoop-bucket) a partir de los checksums de un release.
#
# Uso:   scripts/update-scoop.sh vX.Y.Z
# Requisitos: gh autenticado; para el push, TAP_GITHUB_TOKEN (PAT) o gh.
set -euo pipefail

VERSION_TAG="${1:?uso: update-scoop.sh vX.Y.Z}"
VERSION="${VERSION_TAG#v}"
REPO="AlexCas/omtube"
# NOTA: Cambiar este nombre si Alex crea el repositorio con un nombre distinto.
SCOOP_REPO="AlexCas/scoop-omusic"
WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# Descarga checksums.txt del release.
gh release download "$VERSION_TAG" --repo "$REPO" --pattern checksums.txt --dir "$WORKDIR"

sha() { grep "omusic_${VERSION}_$1.tar.gz" "$WORKDIR/checksums.txt" | awk '{print $1}'; }
SHA_WINDOWS_AMD64="$(sha windows_amd64)"
SHA_WINDOWS_ARM64="$(sha windows_arm64)"

base="https://github.com/${REPO}/releases/download/${VERSION_TAG}"

# Clona el bucket
if [ -n "${TAP_GITHUB_TOKEN:-}" ]; then
  git clone "https://x-access-token:${TAP_GITHUB_TOKEN}@github.com/${SCOOP_REPO}.git" "$WORKDIR/bucket"
else
  gh repo clone "$SCOOP_REPO" "$WORKDIR/bucket"
fi

mkdir -p "$WORKDIR/bucket/bucket"
cat > "$WORKDIR/bucket/bucket/omusic.json" <<EOF
{
    "version": "${VERSION}",
    "description": "Reproductor de música TUI que usa YouTube vía yt-dlp y mpv",
    "homepage": "https://github.com/${REPO}",
    "license": "MIT",
    "depends": [
        "mpv",
        "yt-dlp"
    ],
    "architecture": {
        "64bit": {
            "url": "${base}/omusic_${VERSION}_windows_amd64.tar.gz",
            "hash": "${SHA_WINDOWS_AMD64}",
            "bin": "omusic.exe"
        },
        "arm64": {
            "url": "${base}/omusic_${VERSION}_windows_arm64.tar.gz",
            "hash": "${SHA_WINDOWS_ARM64}",
            "bin": "omusic.exe"
        }
    }
}
EOF

git -C "$WORKDIR/bucket" add bucket/omusic.json
if git -C "$WORKDIR/bucket" diff --cached --quiet; then
  echo "El manifiesto ya está al día para omusic ${VERSION}; nada que publicar."
  exit 0
fi

git -C "$WORKDIR/bucket" \
  -c user.name="omusic-release-bot" \
  -c user.email="omusic-release-bot@users.noreply.github.com" \
  commit -m "omusic ${VERSION}"
git -C "$WORKDIR/bucket" push
echo "Manifiesto de Scoop publicado: omusic ${VERSION} -> ${SCOOP_REPO}"
