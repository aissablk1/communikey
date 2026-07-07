# communikey — ce qui reste, et pourquoi (honnêteté §29)

État au 2026-07-08. Ce fichier liste ce qui n'est **pas** encore livré, avec la **raison
réelle** (souvent un blocage physique : besoin d'un appareil, d'un autre runtime, ou de
vraies données à observer) et le **plan**. Rien ici n'est fait à moitié et présenté comme
fini.

## Livré et MERGÉ sur `main` (2026-07-08, local — non poussé)

> État courant exhaustif : voir `docs/HANDOFF.md`. Entrée cross-provider : `AGENTS.md`.

Ces travaux sont **verts et fusionnés sur `main`** (local ; **push/publication = feu vert
d'Aïssa, non fait**) :

- **Catalogue multi-provider de modèles** (`feat/model-provider-phase1`) : `communikey
  model presets|add` — ~45 providers vérifiés (25 portés du dépôt clawcodex + ~20
  mainstream `[à valider]`), routing « smart » par protocole, **adaptateur natif
  Anthropic** (API Messages, sert anthropic + minimax), `docs/examples/models.json`.
- **5 adaptateurs CLI d'agents de code** (`feat/cli-agent-adapters`) : aider, goose,
  opencode, crush, qwen-code — calibrés sur source primaire, garanties safety-first,
  abstention shell/Claude testée.
- **MCP de contrôle des navigateurs IA — Phase 1** (`feat/browser-mcp`) : sous-projet
  Node `mcp-browsers/` (registre de 14 navigateurs + CDP + 9 outils d'automatisation
  générique). `browser_ai_ask` (IA native) = Phase 2 (bloquée, voir plus bas).

## Déjà résolu (vérifié 2026-07-08)

- Toolchain **Go 1.25** (`go.mod`) · **Actions CI épinglées par SHA complet** ·
  tests dédiés **`relations.go`** (link/unlink/tree) · **externalisation des patterns
  provider** (`providerconfig.go` + `providers.json`) · **TLS 1.3 hybride PQC +
  auth mutuelle** (`serve --tls --authz`) · relicence **Apache-2.0** · config
  **goreleaser** moderne (`formats:` pluriel ; `brews` gardé volontairement pour
  Linux+macOS, cask = macOS-only).

## Reporté par décision d'ingénierie (faible ROI maintenant)

- **Backend GNU `screen`.** L'injection (`screen -X stuff`) marcherait, mais la **lecture
  d'écran** de screen passe par `hardcopy -p` vers un fichier temporaire (pataud, peu fiable
  pour la détection d'état), et screen n'expose pas proprement les titres de fenêtres en CLI.
  tmux (livré) couvre le multiplexeur Unix dominant. Le contrat `Backend` est en place →
  screen = **un struct de plus** le jour où le besoin est réel. Non bâclé exprès.

## Bloqués par un prérequis physique (non testables en session headless)

- **Passkey WebAuthn / FIDO2 (déverrouillage du vault).** Une cérémonie WebAuthn exige un
  **authentificateur** (navigateur, clé matérielle, ou API OS CTAP) et un *relying party* —
  impossible à exécuter/tester dans un CLI sans interface. *Plan* : intégrer `libfido2` /
  les API OS ; dériver la clé du vault depuis l'extension **PRF** de WebAuthn. En attendant,
  le vault est chiffré AES-256-GCM, clé via Argon2id (RFC 9106, passphrase fichier/env, §38).

- **Durcissement réseau au-delà de l'auth mutuelle** (rate-limiting, exposition Internet
  large). L'auth mutuelle TLS est livrée ; le reste n'est **pas cadré** — attend une
  décision de périmètre d'Aïssa avant d'être conçu.

- **Clients mobiles (iOS / iPadOS / Android).** Runtime distinct (app/PWA). Rappel honnête :
  le mobile ne peut **pas** injecter au clavier (sandbox) → il rejoint le bus comme **client**
  (envoyer / relever / approuver / être notifié, façon omnara). *Plan* : client mince vers le
  réseau communikey (`serve`).

- **Windows natif (injection clavier).** Physiquement impossible : pas de PTY maître
  partageable, ConPTY ne s'attache pas à un process déjà lancé. Windows reste **coop-only**
  (inbox). C'est une limite assumée, pas un manque.

## Bloqués par l'absence de vraies données (ne pas inventer, §2/§29)

- **Adaptateurs d'état (codex / gemini / antigravity / clawcodex + aider / goose /
  opencode / crush / qwen-code) — confirmation par capture live.** Tous calibrés sur
  **source primaire** (bundle JS, dépôt officiel, extraction `strings`, code TUI réel),
  mais **aucun confirmé sur un vrai écran capturé en direct** (comme
  `testdata/claude_idle_real.txt`) — Codex absent du PATH, Gemini CLI individuel retiré,
  les autres exigent une session interactive complète (OAuth/setup). *Plan* : capturer un
  vrai écran de chacun (pseudo-tty, comme `communikey teams` l'a fait), lever les CAVEATS
  documentés dans `adapters.go`. Restent `provisoire` dans `provider list` d'ici là.

- **Vérification bout-en-bout HuggingFace Inference API** (catalogue modèle). Seul **Ollama**
  a été testé manuellement. *Bloqué en session* : pas de clé HF disponible et
  `router.huggingface.co` hors de l'allowlist réseau du sandbox. *Plan* : tester avec une
  vraie clé + réseau avant de considérer la mention CHANGELOG comme pleinement prouvée.

- **Browser MCP Phase 2 — `browser_ai_ask` (IA native des navigateurs).** Chaque recette
  d'invocation (chat de Dia, Comet, Brave Leo, Atlas…) doit être **calibrée sur l'UI réelle**
  du navigateur (DOM/sélecteurs), jamais inventée. *Bloqué* : exige le navigateur installé
  et lancé pour inspecter son interface. *Plan* : calibrer un par un les navigateurs
  qu'Aïssa a installés.

- **Bridge Agent Teams.** La livraison dans la *mailbox* d'Agent Teams (`router.go` réserve
  la voie `ChannelBridge`) exige d'en connaître le **format exact** (`~/.claude/teams/…`),
  non documenté publiquement. *Plan* : inspecter une session Agent Teams réelle, puis écrire
  le writer — sans deviner le format.

- **Audit cryptographique externe.** Nécessite un tiers humain/cabinet, pas du code. À faire
  AVANT toute communication « production / Signal-grade » (sinon survente §34).

## En attente d'une décision d'Aïssa (pas une question technique)

- **Merge des branches ci-dessus sur `main`** (via `superpowers:finishing-a-development-branch`).
- **Browser MCP Phase 3** : intégration Go (`communikey browsers`) + entrée dans le
  `~/.claude/settings.json` **global** (config sensible → feu vert requis).
- **Publication** : repo GitHub public, première Release (GoReleaser), tap Homebrew,
  déploiement du site. Voir `docs/PUBLISHING.md`. Rien n'a été poussé sans accord.
- **Clarifier « claude.ai »** de la demande initiale — traité comme équivalent à Claude
  Code (provider `claude` calibré). Si l'intention était le chat web, ce serait une feature
  différente.

**Auteur** : Aïssa BELKOUSSA
