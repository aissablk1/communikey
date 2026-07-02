#!/bin/sh
# communikey — installeur. POSIX sh, auditable (lis-le avant de l'exécuter).
#   curl -fsSL https://communikey.dev/install.sh | sh
# Télécharge le binaire pré-construit correspondant à ta plateforme depuis les
# GitHub Releases, le rend exécutable et le place dans ~/.local/bin.
# Aucun sudo, aucune télémétrie, aucune dépendance autre que curl/tar.
set -eu

REPO="aissablk1/communikey"
BINDIR="${COMKEY_BINDIR:-$HOME/.local/bin}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "communikey: architecture non supportée: $arch" >&2; exit 1 ;;
esac
case "$os" in
  darwin|linux) ;;
  *) echo "communikey: OS non supporté par le binaire ($os). Essaie: go install github.com/$REPO@latest" >&2; exit 1 ;;
esac

asset="communikey_${os}_${arch}.tar.gz"
url="https://github.com/$REPO/releases/latest/download/$asset"

echo "communikey: téléchargement $os/$arch…"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
if ! curl -fsSL "$url" -o "$tmp/communikey.tar.gz"; then
  echo "communikey: aucune release trouvée. Installe depuis la source :" >&2
  echo "  go install github.com/$REPO@latest" >&2
  exit 1
fi
tar -xzf "$tmp/communikey.tar.gz" -C "$tmp"
mkdir -p "$BINDIR"
mv "$tmp/communikey" "$BINDIR/communikey"
chmod +x "$BINDIR/communikey"
ln -sf communikey "$BINDIR/comkey"   # alias court

echo "✓ communikey installé : $BINDIR/communikey (alias : comkey)"
case ":$PATH:" in
  *":$BINDIR:"*) ;;
  *) echo "  Ajoute $BINDIR à ton PATH :  export PATH=\"$BINDIR:\$PATH\"" ;;
esac
"$BINDIR/communikey" help >/dev/null 2>&1 && echo "  Prêt. Lance : communikey help"
