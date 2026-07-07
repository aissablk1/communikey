# HANDOFF.md — état courant exhaustif de communikey

> Mémoire-repo vivante (§40). **Mise à jour à chaque fin de session** par tout agent, tout
> provider. Faits vérifiés, jamais devinés (§2/§29). Lire `AGENTS.md` avant ce fichier.
>
> **Dernière mise à jour** : 2026-07-08 (~02h00) — session « catalogue modèle + adaptateurs
> CLI + MCP navigateurs + consolidation main ».

---

## 0. État de `main` en un coup d'œil

- **Branche `main`** : verte (`go vet`/`build`/`test` OK ; MCP registre 4/4). Tout le travail
  de la session 2026-07-08 y est **mergé** (local ; **non poussé** — gate publication).
- **Aucune branche feature en attente** (les 4 branches de session ont été fusionnées).
- Une **session parallèle** a aussi committé sur `main` (méthode « Ensemble » = flottes
  d'agents cross-vendor : `docs/superpowers/specs/2026-07-08-ensemble-method-design.md`).
  Son travail est **docs-only et disjoint** du mien — préservé intact (§7).

## 1. Livré à la session 2026-07-08 (commits sur `main`)

### Pilier « client multi-provider de modèles » (couche ModelProvider)
- **Catalogue de ~45 providers** : `modelpresets.go` (25 base_urls portées verbatim du dépôt
  `agentforce314/clawcodex`, vérifiées source ; ~20 mainstream marqués `[à valider]`).
- **Adaptateur natif Anthropic** : `modelclient_anthropic.go` (API Messages ; sert `anthropic`
  ET `minimax`). Routing « smart » par `kind` dans `buildModelRegistry` (`modelprovider.go`).
- **Commandes** : `communikey model presets` (catalogue) + `model add <provider>` (écrit
  l'entrée avec le bon `kind` dans `models.json`, idempotent). Helpers `saveModelSpecs`/
  `upsertModelSpec`/`presetToSpec`.
- **`docs/examples/models.json`** : fichier prêt à copier (généré, jamais écrit à la main).
- Tests : `modelpresets_test.go`, `modelclient_anthropic_test.go` — verts.
- Commits : `merge(model)` `27496cd` (= `0680454` + `c9c90d3`).

### Pilier « détection d'état de CLI d'agents » (couche Provider)
- **5 nouveaux adaptateurs** dans `adapters.go` : **aider, goose, opencode, crush,
  qwen-code**, calibrés sur source primaire (dépôts officiels, 2026-07-08), enregistrés dans
  `provider.go`. Tests + fixtures dans `adapters_lot2_test.go` (abstention shell/Claude,
  confirm>busy, idle double-signal). Verts.
- Détails de calibration (tokens réels par agent) : en tête de fonction dans `adapters.go`.
- Commit : `merge(provider)` `b74fa10` (= `d6308f5`).

### Pilier « MCP navigateurs IA » (sous-projet `mcp-browsers/`)
- **Phase 1** : registre 14 navigateurs (`src/browsers.js`) + CDP localhost-strict
  (`src/cdp.js`) + 9 outils (`src/tools.js`) + serveur MCP (`src/index.js`). Tests registre
  4/4 ; smoke `tools/list` OK.
- **Activé** : `npm install` fait dans `main/mcp-browsers` ; déclaré dans
  `~/.claude/settings.json` → `mcpServers.communikey-browsers` (backup :
  `settings.json.bak-2026-07-08-browsermcp`). **Nécessite un redémarrage de Claude Code**
  pour se charger.
- Registre corrigé (§29) : **Dia `verified:true`** (chemin `/Applications/Dia.app/Contents/
  MacOS/Dia` + bundle `company.thebrowser.dia` confirmés sur la machine).
- Commits : `docs(spec)` `4e149b5`, `feat` `51f3fea`, `merge(browser-mcp)` `91bb963`, `fix`
  `08c7dd6`.

### Docs & backlog
- `docs/NEXT.md` rafraîchi · `docs/dev-notes.md` (piège worktree.baseRef) · journal
  `docs/sessions/2026-07-08_01h25_…md` · `merge(docs)` `32f2279`.

## 2. Bloqué — avec la raison réelle (ne jamais masquer, §2/§29)

| Bloqué | Raison réelle | Plan |
|---|---|---|
| **Confirmation capture LIVE des adaptateurs** (codex/gemini/antigravity/clawcodex + aider/goose/opencode/crush/qwen-code) | CLIs non lançables en session headless : Codex hors PATH, Gemini CLI retiré, autres exigent OAuth/setup interactif | Capturer un vrai écran (pseudo-tty) puis lever les CAVEATS ; restent `provisoire` d'ici là |
| **HuggingFace Inference API bout-en-bout** | Pas de clé HF + `router.huggingface.co` hors allowlist réseau du sandbox | Tester avec vraie clé + réseau |
| **Browser MCP Phase 2 (`browser_ai_ask`)** | L'IA de Chrome/Dia/Comet vit dans le **chrome du navigateur, pas le DOM d'une page** ; les cibles CDP *page* ne l'atteignent pas | Rétro-ingénierie CDP **browser-level** par navigateur (Dia = candidat le plus prometteur, natif Chromium installé) — ne pas fabriquer de recette inventée |
| **Bridge Agent Teams** | Format de mailbox `~/.claude/teams/…` non documenté | Inspecter une vraie session avant d'écrire le writer |
| **Passkey WebAuthn (vault)** | Exige un authentificateur physique | `libfido2`/API OS + extension PRF |
| **Audit crypto externe** | Nécessite un tiers humain/cabinet | Avant toute comm « production » |

## 3. En attente d'une décision d'Aïssa (pas technique)

- **Push / PR public + publication** (repo public, Release GoReleaser, tap Homebrew, site) —
  gate explicite ; **rien n'est poussé**. Voir `docs/PUBLISHING.md`.
- **Browser MCP Phase 2** : tenter la rétro-ingénierie CDP de l'IA native de **Dia** ?
  (fragile, non garanti).
- **Sous-commande Go `communikey browsers`** (confort — lancer/déclarer le MCP) : non faite
  (la déclaration `settings.json` suffit à activer). À ajouter si demandé.
- **Valider les 20 providers `[à valider]`** du catalogue · **durcissement réseau**
  (rate-limiting) · clarifier **« claude.ai »** (traité = Claude Code ; si chat web = autre feature).

## 4. Déjà résolu (vérifié 2026-07-08 — ne pas refaire)

Go 1.25 · Actions CI épinglées par SHA complet · tests `relations.go` (link/unlink/tree) ·
patterns provider externalisés (`providerconfig.go` + `providers.json`) · TLS 1.3 hybride
PQC + auth mutuelle (`serve --tls --authz`) · relicence Apache-2.0 · goreleaser moderne
(`formats:` pluriel ; `brews` gardé volontairement, cask = macOS-only) · Phase 1
model-provider (mergée par une session antérieure).

## 5. Comment continuer (guide de reprise)

1. Lire `AGENTS.md` → ce fichier → `docs/NEXT.md`.
2. Build/test vert avant de toucher quoi que ce soit (`go test ./...`).
3. Prendre une tâche **non bloquée** ci-dessus ; si bloquée, passer à la suivante et **noter
   le blocage** dans le journal.
4. Adaptateur d'état nouveau/confirmé → suivre la calibration §2/§29 (source primaire, jamais
   inventer), pattern de `adapters.go` + tests dans un `*_test.go`.
5. Provider LLM à valider → vérifier base_url + modèle sur la doc officielle, passer
   `Src:"known"` → contenu confirmé, retirer `[à valider]`.
6. Fin de session → mettre à jour ce fichier + `docs/NEXT.md` + journal daté.

**Auteur** : Aïssa BELKOUSSA
