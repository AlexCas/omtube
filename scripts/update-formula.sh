#!/usr/bin/env bash
# Regenera y publica Formula/omusic.rb en el tap (AlexCas/homebrew-tap) a partir
# de los checksums de un GitHub Release ya publicado.
#
# Uso:   scripts/update-formula.sh v0.1.0
# Requisitos: gh autenticado (o GH_TOKEN/GITHUB_TOKEN); para el push al tap,
#             TAP_GITHUB_TOKEN (PAT con scope repo) o gh con acceso de escritura.
set -euo pipefail

VERSION_TAG="${1:?uso: update-formula.sh vX.Y.Z}"
VERSION="${VERSION_TAG#v}"
REPO="AlexCas/omtube"
TAP_REPO="AlexCas/homebrew-tap"
WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# Descarga checksums.txt del release.
gh release download "$VERSION_TAG" --repo "$REPO" --pattern checksums.txt --dir "$WORKDIR"

sha() { grep "omusic_${VERSION}_$1.tar.gz" "$WORKDIR/checksums.txt" | awk '{print $1}'; }
SHA_LINUX_AMD64="$(sha linux_amd64)"
SHA_LINUX_ARM64="$(sha linux_arm64)"
SHA_DARWIN_AMD64="$(sha darwin_amd64)"
SHA_DARWIN_ARM64="$(sha darwin_arm64)"

base="https://github.com/${REPO}/releases/download/${VERSION_TAG}"

# Clona el tap (con token si está disponible, p. ej. en CI).
if [ -n "${TAP_GITHUB_TOKEN:-}" ]; then
  git clone "https://x-access-token:${TAP_GITHUB_TOKEN}@github.com/${TAP_REPO}.git" "$WORKDIR/tap"
else
  gh repo clone "$TAP_REPO" "$WORKDIR/tap"
fi

mkdir -p "$WORKDIR/tap/Formula"
cat > "$WORKDIR/tap/Formula/omusic.rb" <<RUBY
class Omusic < Formula
  desc "Reproductor de música TUI que usa YouTube vía yt-dlp y mpv"
  homepage "https://github.com/AlexCas/omtube"
  version "${VERSION}"
  license "MIT"

  depends_on "mpv"
  depends_on "yt-dlp"

  on_macos do
    on_arm do
      url "${base}/omusic_${VERSION}_darwin_arm64.tar.gz"
      sha256 "${SHA_DARWIN_ARM64}"
    end
    on_intel do
      url "${base}/omusic_${VERSION}_darwin_amd64.tar.gz"
      sha256 "${SHA_DARWIN_AMD64}"
    end
  end

  on_linux do
    on_arm do
      url "${base}/omusic_${VERSION}_linux_arm64.tar.gz"
      sha256 "${SHA_LINUX_ARM64}"
    end
    on_intel do
      url "${base}/omusic_${VERSION}_linux_amd64.tar.gz"
      sha256 "${SHA_LINUX_AMD64}"
    end
  end

  def install
    bin.install "omusic"
  end

  def caveats
    <<~EOS
      Para el panel de portada instalá chafa (opcional):
        brew install chafa
    EOS
  end

  test do
    assert_match "omusic", shell_output("#{bin}/omusic --version")
  end
end
RUBY

git -C "$WORKDIR/tap" add Formula/omusic.rb
if git -C "$WORKDIR/tap" diff --cached --quiet; then
  echo "La fórmula ya está al día para omusic ${VERSION}; nada que publicar."
  exit 0
fi
git -C "$WORKDIR/tap" \
  -c user.name="omusic-release-bot" \
  -c user.email="omusic-release-bot@users.noreply.github.com" \
  commit -m "omusic ${VERSION}"
git -C "$WORKDIR/tap" push
echo "Fórmula publicada: omusic ${VERSION} -> ${TAP_REPO}"
