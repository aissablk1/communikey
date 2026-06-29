#!/usr/bin/env bash
# Démo : DEUX agents collaborent via csend — voie coopérative, sans cmux/tmux,
# donc reproductible sur tout OS. Prouve la boucle « un agent écrit, l'autre reçoit
# via son hook ». Aucune télémétrie, store jetable.
set -euo pipefail
cd "$(dirname "$0")/.."

BIN="$(mktemp -d)/csend"
go build -o "$BIN" .
STORE="$(mktemp -d)"
export CSEND_STORE_DIR="$STORE"
echo "Store de démo : $STORE"
echo

echo "1) alice et bob rejoignent le bus (n'importe quel terminal / OS / provider)"
CSEND_AGENT_ID=alice "$BIN" register --provider bash   >/dev/null
CSEND_AGENT_ID=bob   "$BIN" register --provider codex  >/dev/null
"$BIN" agents
echo

echo "2) alice écrit à bob"
CSEND_AGENT_ID=alice "$BIN" inbox bob "bob, le build est vert — à toi de déployer"
echo

echo "3) bob 'reçoit' (exactement ce que ferait son hook UserPromptSubmit)"
CSEND_AGENT_ID=bob "$BIN" hook
echo

echo "4) bob relance son hook → silencieux (déjà consommé)"
CSEND_AGENT_ID=bob "$BIN" hook
echo "   (rien ci-dessus = correct)"
echo

echo "Démo OK. Pour la réception live permanente : câble 'csend hook' (csend hook --install)."
rm -rf "$STORE" "$(dirname "$BIN")"
