# AGENTS.md — contexte pour agents IA (tous providers, tous types)

> **Lis ce fichier EN PREMIER.** Point d'entrée cross-provider (Claude, Codex, Gemini,
> Droid, Cursor, humains…). Format `AGENTS.md` standard multi-outils. But : que
> n'importe quel agent futur comprenne le projet et **continue le travail sans casser
> ce qui existe** ni **réinventer** ce qui est déjà là.
>
> **Auteur / propriétaire** : Aïssa BELKOUSSA. Crédit d'auteur = Aïssa BELKOUSSA seul
> (aucun co-auteur IA dans commits/docs). Commits : conventionnels, en français.

---

## 1. Ce qu'est communikey

Outil **Go** (binaire unique, package `main`, module `github.com/aissablk1/communikey`,
**Go 1.25**, licence **Apache-2.0**) avec **trois piliers** :

1. **Bus de messagerie inter-sessions** (le cœur d'origine) : injecte un message dans une
   **autre** session d'agent CLI en cours (state-aware : lit l'écran de la cible avant
   d'agir ; chiffré ; cross-workspace via le socket cmux). Fichiers : `bus.go`, `cmux.go`,
   `inbox.go`, `relations.go`, `net.go`, `tlsbus.go`, `router.go`, `state.go`.
2. **Détection d'état de CLI d'agents** (`Provider`) : classe l'écran d'un CLI agent en
   idle / busy / confirm / unknown, **safety-first** (jamais d'auto-submit dans une
   confirmation ou un shell nu). Fichiers : `provider.go`, `adapters.go`,
   `providerconfig.go`. Adaptateurs : claude, codex, gemini, antigravity, clawcodex,
   **aider, goose, opencode, crush, qwen-code** (les 5 derniers ajoutés le 2026-07-08).
3. **Client multi-provider de modèles** (`ModelProvider`) : appelle un backend LLM
   déclaré dans `~/.claude/communikey/models.json`. `communikey model presets|add|list|
   test|call|secret set`. **Catalogue de ~45 providers** (`modelpresets.go`) + adaptateur
   **openai-compatible** (`modelclient_openai.go`) + adaptateur **natif Anthropic**
   (`modelclient_anthropic.go`, sert `anthropic` et `minimax`). Fichiers : `model*.go`.

**Sous-projet séparé** : `mcp-browsers/` — **serveur MCP Node/ESM** de contrôle des
**navigateurs IA** (Dia, Comet, Atlas, Brave Leo, Chrome/Gemini…) via CDP. Voir §5.

## 2. Build & test (vérifier vert AVANT tout commit)

```bash
# Go (binaire principal) — package main unique : un fichier qui ne compile pas casse tout
go vet ./...
go build -o /dev/null ./...      # NB: un binaire `communikey` présent dans le dossier fait
                                 # échouer `go build ./...` sans -o (quirk cosmétique)
go test ./...                    # httptest uniquement, aucun appel réseau réel

# MCP navigateurs (Node)
cd mcp-browsers && npm install && npm test    # test registre (sans navigateur)
```

## 3. Conventions STRICTES (ne pas violer)

- **Zéro dépendance externe dans le binaire Go** (stdlib seulement). Les deps vont dans le
  sous-projet Node `mcp-browsers/` (isolé).
- **Calibration des adaptateurs d'état (§2/§29)** : les tokens de détection (busy/confirm/
  idle) sont **VÉRIFIÉS SUR SOURCE PRIMAIRE** (dépôt/bundle/binaire réel), **jamais
  inventés**. Un adaptateur non confirmé sur écran live reste `provisoire`. Invariant de
  sûreté : idle exige un **double signal** (boîte vide + footer distinctif) ; confirm > busy
  > idle > unknown ; ne jamais lire un shell nu ou un écran Claude comme idle.
- **Catalogue de modèles** : les base_urls portées de `agentforce314/clawcodex` sont
  vérifiées à la source ; les autres sont marquées `[à valider]` (`Src:"known"`). Ne pas
  présenter une base_url ou un modèle non vérifié comme certain.
- **Sécurité** : jamais de secret côté client ; port de debug CDP **localhost strict** ;
  vault AES-256-GCM + Argon2id. Voir `SECURITY.md`, `docs/THREAT-MODEL.md`.
- **Git multi-sessions** : plusieurs sessions tournent en parallèle. **Jamais `git add -A`/
  `-u`/`.`** ni `git commit -a`. **Stage fichier par fichier par chemin explicite.** Une
  session ne doit jamais écraser le travail d'une autre (incident fondateur). Sur `main`,
  brancher de préférence.
- **Vert avant commit** : ne jamais committer du code cassé.

## 4. Protocole de journalisation CONTINUE (le « pour toujours » — OBLIGATOIRE)

Pour que le contexte survive entre sessions, providers et types d'agents, **chaque agent
qui travaille ici DOIT** :

1. **Au démarrage** : lire `AGENTS.md` (ce fichier) → `docs/HANDOFF.md` (état courant
   exhaustif) → `docs/NEXT.md` (backlog : reste / bloqué / décisions).
2. **Pendant le travail** : consigner les actions dans un journal daté
   `docs/sessions/AAAA-MM-JJ_HHhMM_<slug>.md` (méthode QQOQCCP : Qui/Quoi/Où/Quand/Comment/
   Combien/Pourquoi + Actions analysées / réalisées / à mener / Notes). Un jalon = une entrée.
3. **À chaque tâche terminée** : vérifier vert → **commit atomique** (message clair, FR
   conventionnel, chemins explicites) → noter l'action dans le journal.
4. **En fin de session** : mettre à jour `docs/HANDOFF.md` (état courant) et `docs/NEXT.md`
   (reste/bloqué/décisions). **Ne jamais masquer un blocage** : si une tâche est bloquée,
   la noter explicitement avec sa **raison réelle**, jamais inventer une solution de façade.

> Claude Code automatise partiellement l'étape 2 via un hook de journal de session. Les
> autres providers suivent le protocole **manuellement**. Le repo est la mémoire commune
> (aucune dépendance à une mémoire propriétaire d'un provider).

## 5. MCP navigateurs (`mcp-browsers/`)

Serveur MCP stdio (Node/ESM) : registre de 14 navigateurs IA (`src/browsers.js`) + CDP
(`src/cdp.js`, localhost strict) + 9 outils (`src/tools.js` : list/launch/attach/navigate/
read/click/fill/screenshot/eval). Déclaré dans `~/.claude/settings.json` →
`mcpServers.communikey-browsers`. Design : `docs/superpowers/specs/2026-07-08-communikey-
browser-mcp-design.md`. **Phase 2** (`browser_ai_ask` — invoquer l'IA native) : **bloquée**
— l'IA de ces navigateurs vit dans le *chrome* du navigateur, pas le DOM d'une page ; les
cibles CDP *page* ne l'atteignent pas → nécessite de la rétro-ingénierie CDP browser-level
par navigateur (à ne pas fabriquer à l'aveugle).

## 6. Où trouver quoi

| Besoin | Fichier |
|---|---|
| **État courant exhaustif** (ce qui est fait/bloqué/à décider) | `docs/HANDOFF.md` |
| **Backlog** (reste, raisons de blocage, décisions attendues) | `docs/NEXT.md` |
| **Journaux de session** (historique détaillé, horodaté) | `docs/sessions/*.md` |
| Specs de design | `docs/superpowers/specs/*.md` |
| Sécurité / modèle de menace | `SECURITY.md`, `docs/THREAT-MODEL.md` |
| Historique des changements | `CHANGELOG.md` |
| Pièges de dev rencontrés | `docs/dev-notes.md` |
| Publication (en attente feu vert Aïssa) | `docs/PUBLISHING.md` |

**Auteur** : Aïssa BELKOUSSA
