---
title: "Hermes Agent (Nous Research) — clarification de nom + brief de calibration d'adaptateur"
date: 2026-07-08
auteur: Aïssa BELKOUSSA
statut: recherche vérifiée (source primaire) ; adaptateur PAS encore écrit (voir §5 garde-fous)
tags: [communikey, hermes, nous-research, provider, adaptateur, calibration]
---

# Hermes Agent (Nous Research) — clarification + calibration

> Vérifié sur **source primaire** le 2026-07-08 (`github.com/NousResearch/hermes-agent`).
> Le non-vérifié est explicitement flagué (§29). Ce fichier sert de **brief de calibration**
> pour écrire l'adaptateur d'état — mais l'écriture dans `adapters.go` est **différée** (§5).

## 1. Verdict de la collision de nom — UN SEUL PRODUIT

Le panneau cmux « **Hermes** », le MCP `hermes` de l'environnement d'Aïssa, et **Hermes Agent
de Nous Research** sont **le même produit**. Le « bridge de messagerie » (Telegram / Discord /
Slack / WhatsApp / Signal) **est la *gateway* de Hermes Agent** (`hermes gateway` = "Start the
messaging gateway"), pas un homonyme. Le MCP `hermes` = la couche messagerie de Hermes Agent
exposée via MCP.
→ Confirmé : `providerconfig.go` (entrée `hermes`, jadis « non confirmé visuellement ») peut
passer à **confirmé** quand ce fichier sera libre d'édition concurrente (§5).
⚠️ **Matrix** figure dans la description du MCP local mais **PAS** dans la liste plateformes du
README (Telegram/Discord/Slack/WhatsApp/Signal/Email/Home Assistant) — Matrix **non confirmé**.

## 2. Ce qu'est Hermes Agent (vérifié)

Agent **autonome auto-améliorant** à interface terminal (TUI réelle) + gateway messagerie + cron
+ subagents + 40+ outils. **Model-agnostic** (« Use any model you want » — pas lié aux poids
Hermes). Auto-description README : « The self-improving AI agent built by Nous Research… the only
agent with a built-in learning loop — creates skills from experience… builds a deepening model of
who you are across sessions. » **Général** (le code est *une* capacité, pas tout le produit).
- Repo : `github.com/NousResearch/hermes-agent` · **Licence MIT** (© 2025 Nous Research) · ~82 % Python + TS.
- Install : `curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash` (installeur git ;
  uv + Python 3.11 + Node). ⚠️ `curl|bash` : à auditer avant exécution (§19/§38).

## 3. Données de calibration (⚠️ mauvais front-end pour l'install d'Aïssa — voir mise à jour)

> **MISE À JOUR 2026-07-08 (front-end CONFIRMÉ, §29)** : diagnostic read-only de l'install
> d'Aïssa → `~/.local/bin/hermes` exec `~/.hermes/hermes-agent/venv/bin/hermes`, point d'entrée
> Python `from hermes_cli.main import main`, version **`hermes-agent 0.14.0`**. Son front-end réel
> est le **CLI Python `hermes_cli`**, PAS le `ui-tui` TS. **Les tokens ci-dessous (extraits du
> `ui-tui`) ne s'appliquent donc PAS à son install** — conservés seulement comme référence de l'UI
> TS. La calibration correcte exige une **capture d'écran live du TUI Python `hermes_cli`** (§5) ;
> la doctrine du projet (AGENTS.md §3) impose de toute façon la capture réelle, pas les strings du
> source (composées/stylées au rendu). Adaptateur = **bloqué** tant que cette capture n'existe pas.

### Calibration Python `cli.py` — DÉFINITIVE (source 0.18.2 en ligne + cross-check 0.14.0 local)

**Dispatch** : `hermes` nu → `cmd_chat` → défaut `display.interface = "cli"` → REPL **Python
`cli.py`** (prompt_toolkit). Le node `ui-tui` (ink) ne se lance QUE sur opt-in (`--tui`,
`HERMES_TUI=1`, `display.interface: tui`). En **0.14.0**, `_resolve_use_tui` **n'existe pas** →
c'est **toujours** le `cli.py` Python.

**Stabilité vérifiée** : tous les tokens ci-dessous, trouvés dans `main` (0.18.2) en ligne, sont
**présents dans le `cli.py` 0.14.0 local d'Aïssa** (grep, comptes ≥1) → sûrs à calibrer.

**CONFIRM — sécurité critique (l'adaptateur DOIT détecter ceci comme `confirm` et NE JAMAIS
auto-valider)** :
- Titre panneau : **`⚠️  Dangerous Command`** (⚠️ + **deux** espaces).
- Options : **`❯ 1. Allow once`** / **`  2. Allow for this session`** / **`  3. Add to permanent
  allowlist`** / **`  4. Deny`** (+ `Show full command`).
- Indice : **`↑/↓ to select, Enter to confirm`** (+ `  (Ns)`).
- ⚠️ **DANGER** : **Enter valide le choix surligné — défaut = « Allow once » = APPROUVE** ; les
  chiffres `1`–`9`/`0` valident **immédiatement** ; **pas** de binding `y`/`n` dans ce panneau. →
  un outil externe qui tape Entrée ou un chiffre **approuve une commande dangereuse**. Détection
  = non négociable.
- Confirm secondaire (slash destructif `/clear`,`/reset`) : footer `Type 1/2/3 or use ↑/↓ then
  Enter. ESC/Ctrl+C cancels.` — ici `y`/`n` marchent comme texte tapé (≠ panneau dangereux).

**BUSY** :
- Symbole prompt : **`⚕ ❯ `** (style `prompt-working`).
- Placeholder saisie : **`msg=interrupt · /queue · /bg · /steer · Ctrl+C cancel`**.
- Spinner : frames braille `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` + `  {txt}  ({elapsed}s)` ; `Processing command...` ;
  `⚡ Interrupting agent...`.

**IDLE** :
- Symbole prompt : **`❯ `** (SANS `⚕`).
- Placeholder = **vide** (`""`). → idle = `❯ ` **seul** + statut vide + **pas** de spinner.
- Accueil (démarrage uniquement, PAS le prompt live) : `Welcome to Hermes Agent! Type your
  message or /help for commands.`

**Invariant adaptateur** (comme les autres, safety-first) : **confirm > busy > idle > unknown**.
Idle exige le **double signal** : prompt `❯ ` **sans** `⚕` **ET** absence de spinner/`Processing`.
Le panneau `⚠️  Dangerous Command` / options `Allow`/`Deny` = **confirm absolu**. Écran ambigu ou
shell nu → `unknown`, jamais idle.

**Statut calibration** : **SOLIDE** (tokens vérifiés dans la source 0.18.2 + confirmés dans le
0.14.0 local d'Aïssa). Idéalement une **capture d'écran live** confirmera le RENDU exact
(ANSI/couleurs/troncature) avant de lever le flag `provisoire` — mais l'adaptateur est **écrivable
avec confiance**, ce qui n'était pas le cas sur les tokens ui-tui.

### (Référence UI TS `ui-tui` — NE PAS utiliser pour l'install Python d'Aïssa)

**⚠️ CAVEAT DÉCISIF** : ces tokens viennent du **front-end TS `ui-tui`**. Il existe **AUSSI** un
CLI **Python `cli.py`** (`prompt_toolkit`, 743 Ko) dont les strings **n'ont PAS été vérifiés** et
**peuvent différer**. Les commentaires de `prompts.tsx` disent « mirrors the CLI approval panel »
(similaire, pas identique). **Il faut savoir quel front-end le `hermes` d'Aïssa lance avant de se
fier à ces tokens** — surtout la confirmation (§5).

**(a) BUSY** (`verbs.ts`, `appChrome.tsx`, `thinking.tsx`) : gérondif rotatif + `…` — `pondering`,
`synthesizing`, `analyzing`, `reasoning`, `computing`, `formulating`, `brainstorming`… ; verbes
d'outil `browsing`/`reading`/`running`/`writing`/`patching`/`executing`/`deleting`… ; label
littéral **`Thinking`** + spinner braille ; lignes d'outil `● <label> (1.2s)` ; `analyzing tool
output…` / `drafting...`.

**(b) IDLE** (`placeholders.ts`, `appChrome.tsx`) : placeholder de saisie **`Ask me anything…`**
(+ `Try "/help" for commands`) ; horloge idle **`✓ <durée>`**. ⚠️ le **mot de statut idle** exact
(prop runtime) = **NON TROUVÉ** — ne pas en supposer un.

**(c) CONFIRM** (`prompts.tsx`) — **le plus critique** :
- Approbation : header **`⚠ approval required · <description>`** ; options **`1. Allow once`** /
  **`2. Allow this session`** / **`3. Always allow`** / **`4. Deny`** ; footer `↑/↓ select · Enter
  confirm · 1-<N> quick pick · Esc/Ctrl+C deny`.
- Oui/Non : header **`? <title>`** (ou **`⚠ <title>`** si danger) ; lignes **`No`**/**`Yes`** ;
  footer `↑/↓ select · Enter confirm · Y/N quick · Esc cancel`.
- Clarify : heading **`ask <question>`**.
- **Sûreté** : `y`, `Enter`, et les chiffres `1`–`N` **valident tous** une confirmation.

## 4. Positionnement dans Ensemble (SP0)

Hermes Agent est un **agent généraliste auto-améliorant** (pas un CLI de code spécialisé). Dans la
flotte Ensemble, candidat pour un **rôle distinctif** : sa *learning loop* (« crée des skills depuis
l'expérience, modélise l'utilisateur entre sessions ») en fait un bon **membre persistant / mémoire
de continuité**, ou un builder/checker vendeur de plus (diversité cross-vendor = le moat). Caveat :
étant généraliste, ne pas le présenter comme un builder de code spécialisé sans dogfood.

## 5. État & garde-fous (STRICT)

- **✅ Adaptateur ÉCRIT & mergé dans `main`** (`fad068d`, 2026-07-09) : `adapters_hermes.go`
  (`newHermesProvider()`) + enregistrement `provider.go` + `adapters_hermes_test.go` (5 tests verts).
  Écrit sur branche isolée `feat/hermes-adapter` (fichier neuf → **zéro conflit `adapters.go`**, §7),
  fast-forward propre dans `main`. **confirm + busy détectés ; idle NON détecté** (sûr, comme aider —
  prompt `❯` = glyph Claude + placeholder vide + pas de footer persistant → pas de double signal).
- **STATUT `provisoire`** (§2/§29) : calibré sur SOURCE (repo 0.18.2 + cross-check install locale
  0.14.0 → tokens stables), **pas sur capture d'écran live**. Fixtures de test SYNTHÉTIQUES (pas
  `*Real`). Invariant tenu : confirm > busy, jamais d'auto-validation ; écran ambigu → `unknown`.
- **Seul reste pour lever `provisoire`** : une **capture d'écran live** du `cli.py` (lancer `hermes`,
  capturer busy/idle/confirm réels) pour confirmer le RENDU (ANSI/couleurs/troncature) vs les tokens
  source. Cette capture pourrait aussi révéler un footer idle sûr → activer la détection idle.

## 6. Sources & non-vérifié

**Vérifié (fetché)** : [repo](https://github.com/NousResearch/hermes-agent) · README · LICENSE ·
`ui-tui/src/components/prompts.tsx` · `appChrome.tsx` · `thinking.tsx` · `content/verbs.ts` ·
`content/placeholders.ts`.
**NON vérifié** : strings du `cli.py` Python (peuvent différer — décisif) ; mot de statut idle ;
étoiles (~180 k–210 k, conflit) ; « lancé fév. 2026 / fastest-growing » (marketing secondaire ;
le repo est *créé* le 2025-07-22 via l'API) ; Matrix comme plateforme ; lookalikes tiers
(`hermes-agent.org`, `hermes-ai.net`, forks) à ignorer — seul `NousResearch/hermes-agent` fait foi.

---

**Auteur** : Aïssa BELKOUSSA · Apache-2.0
