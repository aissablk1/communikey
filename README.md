# csend

Messagerie **inter-sessions** pour agents CLI (Claude Code & co.) hébergés dans
**cmux / Vibe Island**. `csend` injecte un message dans une **autre** session en
cours — **state-aware** (lit l'écran de la cible avant d'agir) et
**cross-workspace** (n'importe quel workspace, visible ou non, **zéro flicker**,
via le socket JSON-RPC de cmux).

> Ce que les outils natifs ne permettent pas (injecter dans une TUI vivante) et
> que `tmux send-keys` fait à l'aveugle, `csend` le fait proprement : en lisant
> l'état de la cible (idle / busy / confirmation) avant de livrer.

## Commandes

```
csend list                     toutes les sessions agent + leur état
csend tree                     graphe familial (parent → enfants)
csend send <cible> "msg"       message à une session
csend send --down "msg"        broadcast aux enfants de la session courante
csend send --up "msg"          au parent (aussi --to-siblings / --to-descendants)
csend read <cible>             lire l'écran d'une session
csend link <enfant> <parent>   déclarer une parenté ; unlink pour retirer
```

**Cible** : nom de workspace (`SACEM`), session-id (`7f384610`) ou ref (`surface:42`).

## Livraison gouvernée (garde-fous)

- **`--auto`** (défaut) : valide (Entrée) **uniquement si la cible est idle** ;
  **busy** → déposé sans valider ; **confirmation y/N** / prompt inconnu →
  **refusé** ; **jamais** la session courante.
- `--stage` (déposer) · `--send` (valider même si busy) · `--force` (passe outre).
- Audit append-only dans `~/.claude/logs/csend.jsonl` (hash du texte, pas le texte brut).

## Comment ça marche

- **Topologie** via `cmux tree --json`.
- **I/O surface** via le **socket JSON-RPC** de cmux (`surface.read_text`,
  `surface.send_text`, `surface.send_key`), en passant le **bon `workspace_id`**
  de chaque surface — ce qui débloque le cross-workspace (le CLI `cmux` défaute
  le workspace sur l'appelant et échoue hors de son workspace).
- **Détection d'état** : lecture de l'écran (texte brut, pas d'ANSI) → idle /
  busy (`esc to interrupt`) / confirmation (menu numéroté, y/N) — testée sur de
  vrais écrans.

## Build & install

```
make install      # ou: voir install.sh — binaire universel + symlink ~/.local/bin/csend
```

## Stack

Go (binaire unique, zéro dépendance, stdlib uniquement — crypto incluse),
backend cmux (socket Unix).

## Vision v2 — bus inter-agents universel

`csend` évolue vers un **bus de messagerie inter-agents** : cross-session,
cross-provider (Claude / Codex / Gemini…), cross-terminal (bash / cmux / tmux /
Terminal) et cross-OS, avec **mémoire persistante**, **crypto E2E** (PQC-ready) et
**vault à passkey**. Principe directeur : **inbox coopératif = colonne vertébrale
universelle ; injection clavier live = repli Unix**. Windows natif et mobile sont
coopératif-only (pas d'injection clavier — limite physique assumée).

Design & roadmap : `docs/superpowers/specs/2026-06-27-csend-bus-universel-design.md`.

**Auteur** : Aïssa BELKOUSSA · contact@aissabelkoussa.fr
