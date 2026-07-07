# communikey — Bilan complet de session (2026-07-06 → 2026-07-07)

Session longue, plusieurs fils. Ce fichier est le point d'entrée unique : il résume
tout, renvoie vers le détail (commits + `docs/sessions/2026-07-06_communikey-mldsa-argon2id.md`
pour le narratif fin des rounds 1-2), et liste — honnêtement — ce qui reste à faire et
pourquoi.

## Qui / Quoi / Où / Quand / Pourquoi

- **Qui** : Aïssa BELKOUSSA, via cette session Claude Code (répertoire de travail
  principal — **pas** le worktree `.worktrees/model-provider-phase1/` d'une autre
  session, actif en parallèle sur le même dépôt).
- **Où** : `/Volumes/Professionnel/Projets/Développement/Outils/communikey`, branche `main`.
- **Quand** : démarré 2026-07-06 (demande initiale), poursuivi 2026-07-07.
- **Pourquoi** : demande initiale de durcissement crypto de `communikey`, devenue —
  après tri, clarifications et une décision explicite de continuer en autonomie —
  une série de rounds : durcissement post-quantique, réseau, puis détection
  d'agents CLI.

## Chronologie et décisions clés

1. **Demande initiale incohérente** — une liste de 19 « sécurités » à ajouter
   (RSA-1024, MD5, SHA-1, Shor, GNFS, Coppersmith–Winograd, Busy Beaver,
   classification des groupes finis simples, FairPlay, UPX+AES, …). Auditée
   item par item avant tout code : la plupart n'étaient pas des primitives de
   sécurité valides (algorithmes d'attaque, objets mathématiques purs, ou
   régressions type RSA-1024/MD5/SHA-1). Seul **ML-DSA (Dilithium)** était
   légitime et déjà sur la roadmap du projet — expliqué à Aïssa avant d'agir.
2. **Décision explicite** (QCM) : implémenter ML-DSA en **hybride** avec Ed25519
   (comme le KEM existant X25519⊕ML-KEM-768), via `filippo.io/mldsa` (la stdlib Go
   n'a pas encore de `crypto/mldsa` public — vérifié, proposé pour Go 1.27).
3. **Décision explicite** : accepter Argon2id (RFC 9106) comme 2ᵉ dépendance
   externe pour durcir le KDF du vault (PBKDF2 → Argon2id).
4. **« Continue le travail »** (instruction explicite, autonomie sans validation
   pas-à-pas mais blocages signalés) : re-tri du backlog via les fichiers de
   session existants (`docs/NEXT.md`, sessions du 07-05) — constat honnête que
   tout était déjà fait ou réellement bloqué, sauf l'auth mutuelle réseau.
5. **Découverte en cours de route** : Gemini CLI individuel officiellement
   retiré le 18/06/2026 (Google) → pivot vers **Antigravity CLI**, son
   successeur officiel.
6. **Décision explicite** (QCM, respect du plafond §27 anti-yak-shaving) :
   continuer sur une 5ᵉ vague — l'authentification mutuelle TLS.
7. **Deux demandes d'élargissement de providers** (messages reçus en cours de
   travail) : (a) une liste de ~25 backends LLM (image), (b) des CLI d'agents
   dont clawcodex.app, claude.ai, codex. Investigation a montré que (a) est
   déjà en cours de construction par une **autre session active dans un
   worktree isolé** (`.worktrees/model-provider-phase1/`) — non dupliqué, la
   collision aurait été directe. (b) traité : ClawCodex ajouté.

## Ce qui a été FAIT (livré, testé, committé)

| # | Commit | Contenu |
|---|---|---|
| 1 | `24c4227` | Signature hybride Ed25519 ⊕ ML-DSA-65 (`crypto.go`, `authz.go` corrigé au passage) |
| 2 | `eb73942` | Vault Argon2id (RFC 9106) au lieu de PBKDF2 |
| 3 | `dc50db9` | Journal round 1 |
| 4 | `d26f30c` | Adaptateur provider `antigravity` (successeur de Gemini CLI) |
| 5 | `6c9e075` | Authentification MUTUELLE au niveau TLS (`serve --tls --authz`) |
| 6 | `70e9818` | Adaptateur provider `clawcodex` (agentforce314/clawcodex) |
| 7 | `8298f26` | Journal round 2 |

Chaque commit : `go vet` + suite de tests complète + `go build` + `go mod verify`
vérifiés **verts avant** commit (jamais de code cassé committé, §32). Staging par
chemin explicite à chaque fois (jamais `git add -A`), zéro fichier de l'autre
session touché.

## Ce qui RESTE À FAIRE — honnêtement, avec la raison

### Bloqué par une contrainte réelle (pas actionnable par moi maintenant)
- **Confirmation par capture d'écran live** des adaptateurs `codex`, `gemini`,
  `antigravity`, `clawcodex` — tous calibrés sur **source primaire** (dépôt/bundle/
  binaire réels), aucun sur un **écran réellement observé en direct**. Codex absent
  du PATH ; Gemini CLI individuel retiré ; Antigravity et ClawCodex nécessitent une
  session interactive complète (OAuth ou setup). *Reste `provisoire` dans
  `provider list` tant que ce n'est pas levé.*
- **Livraison dans la mailbox Agent Teams** (`router.go` réserve la voie
  `ChannelBridge`) — format jamais observé, écrire un parseur maintenant serait
  deviner un schéma (interdit, §2/§29).
- **Passkey/WebAuthn** (déverrouillage du vault) — exige un authentificateur
  physique, impossible à tester en CLI headless.
- **Audit cryptographique externe** — nécessite un tiers humain/cabinet, pas du code.
- **Backend `screen`, clients mobiles, Windows natif (injection clavier)** —
  ROI faible ou impossibilité technique déjà documentée (`docs/NEXT.md`).

### Réservé à une décision d'Aïssa (pas une question technique)
- **Publication** (repo public, tag de release, tap Homebrew, site) — explicitement
  hors autonomie depuis la session du 07-05.
- **Durcissement réseau au-delà de l'auth mutuelle** (rate-limiting, exposition
  Internet large) — pas commencé, pas cadré.
- **« claude.ai » de la dernière demande** — traité comme équivalent à Claude Code
  (déjà le provider `claude` calibré). Si Aïssa visait autre chose (le chat web,
  par exemple), ce serait une feature complètement différente (pas de la détection
  d'écran terminal) — à clarifier avant d'agir.

### En cours ailleurs — NE PAS dupliquer
- **`communikey model`** (client multi-provider LLM direct : DeepSeek, Anthropic,
  OpenAI, Gemini API, Ollama, vLLM… — la liste des ~25 backends demandée) : déjà
  activement construit par une **autre session**, dans
  `.worktrees/model-provider-phase1/` (`model.go`, `modelprovider.go`,
  `modelprovider_test.go`, méthode SDD avec rapports de tâches, dernière activité
  vue le 2026-07-07 à 10h09). Vérifié via `~/.claude/sessions-active/` avant de
  décider de ne rien construire en parallèle dans le répertoire principal.

## Fichiers de référence
- Détail narratif rounds 1-2 : `docs/sessions/2026-07-06_communikey-mldsa-argon2id.md`
- Backlog honnête (mis à jour à chaque livraison) : `docs/NEXT.md`
- Modèle de menace : `docs/THREAT-MODEL.md` · Politique de sécurité : `SECURITY.md`
- Historique détaillé : `CHANGELOG.md` (section « Non publié — vers 0.3.0 »)

**Auteur** : Aïssa BELKOUSSA
