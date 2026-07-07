# Changelog

Toutes les évolutions notables de communikey sont consignées ici.

Le format suit [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/), et le projet
adopte le [versionnage sémantique](https://semver.org/lang/fr/).

## [Non publié] — vers 0.3.0

### ⚠️ Changements incompatibles (BREAKING)
- **Cryptographie** : le transcript signé + l'AEAD lient désormais la clé publique de
  l'expéditeur ET une AAD applicative `from→to`. Un message scellé par 0.2.0 ne se vérifie
  plus sous 0.3.0 (durcissement anti-replay / anti-ré-emballage). `Seal`/`Open` acceptent une
  AAD variadique optionnelle (appels sans contexte inchangés).
- **Signature hybride Ed25519 ⊕ ML-DSA-65** : `PublicBundle` gagne `MLDSAPub` et
  `SealedMessage` gagne `SenderMLDSAPub`/`MLDSASig` ; `Open` exige désormais que les
  **DEUX** signatures soient valides. Un message scellé sous une version antérieure de
  0.3.0-dev (sans ces champs) ne se vérifie plus. `go.mod` passe de `go 1.24` à
  `go 1.25` (requis par la dépendance ci-dessous).
- **Vault en Argon2id (plus PBKDF2)** : `SealVault`/`OpenVault` dérivent désormais la
  clé AES-256 via Argon2id (RFC 9106) au lieu de PBKDF2-SHA256. Un vault scellé par une
  version antérieure de 0.3.0-dev **ne se déverrouille plus** (échoue comme une mauvaise
  passphrase — aucune migration automatique, alpha assumée).

### Ajouté
- `communikey journal [--json]` : trace du bus (de→à, **hash uniquement**, jamais le clair).
- `communikey key <cible> <touche>` : envoie une touche brute (enter/escape/ctrl+c/ctrl+u…).
- `communikey hook` **provider-aware** : identité auto-dérivée du `session_id` (zéro-config), forme
  de sortie par éditeur (Claude/Codex `hookSpecificOutput` ; Gemini brut), `--install {claude|codex|gemini}`,
  `--provider` pour forcer la forme.
- Enveloppe *provider-aware* : champs `provider`/`kind` sur les messages (collaboration cross-vendor visible).
- `docs/THREAT-MODEL.md` (modèle de menace honnête, crypto **non auditée**), `docs/cross-vendor-setup.md`
  (câbler Claude+Codex+Gemini), `docs/CLA.md`, `NOTICE`.
- Démo `scripts/demo-cross-vendor.sh` (« Green Build Relay » : 3 éditeurs, 1 bus).
- **Providers externalisables sans recompilation** : `~/.claude/communikey/providers.json`
  optionnel (purement additif — claude/codex/gemini restent compilés en dur, inchangés).
  `communikey provider list` (statut calibré/provisoire/personnalisé/absent de chaque
  provider connu) et `communikey provider test <name>` (teste un écran lu depuis stdin —
  la boucle de calibrage communautaire).
- **`communikey teams`** : découverte **lecture seule** des Agent Teams natives de Claude Code
  (`~/.claude/teams/*/config.json`). Schéma confirmé sur un vrai fichier capturé le 2026-07-05
  (pas deviné) ; pas encore de livraison dans la mailbox (format non observé).
- **Adaptateur `antigravity`** (`adapters.go`) : successeur officiel de Gemini CLI pour les
  comptes individuels (Gemini CLI retiré le 18/06/2026 — developers.googleblog.com, vérifié
  2026-07-07). Calibré par extraction statique (`strings`) sur le binaire réellement installé
  (Homebrew cask `antigravity-cli` 1.0.16, `agy`) — pas de capture d'écran live (session OAuth
  Google complète requise). `provider list` le référence en `provisoire`, même statut que
  codex/gemini.
- `communikey model list|test|call` : client multi-provider de modèles via un adaptateur
  générique compatible OpenAI, déclaratif via `~/.claude/communikey/models.json`, secrets
  scellés dans le vault existant (`communikey model secret set`, valeur lue sur stdin).
  Vérifié de bout en bout contre **Ollama** ; LocalAI, HuggingFace Inference API et tout autre
  endpoint compatible OpenAI sont supportés par conception mais pas encore vérifiés en
  conditions réelles. Phase 1 seulement — voir
  `docs/superpowers/specs/2026-07-06-communikey-model-provider-design.md` pour le design complet.
- `communikey model presets` / `communikey model add <provider>` : **catalogue de ~45 providers
  d'inférence** prêts à l'emploi, avec routing « smart » (communikey choisit le protocole `kind`
  par provider). 25 backends portés verbatim du dépôt officiel `agentforce314/clawcodex`
  (base_urls vérifiées à la source), + ~20 mainstream openai-compatibles (Groq, Mistral, xAI,
  Cerebras, Perplexity…, marqués `[à valider]`). Ajoute l'**adaptateur natif Anthropic** (API
  Messages) qui sert `anthropic` **et** `minimax`. `docs/examples/models.json` : fichier prêt à copier.

### Changé
- **Licence** : MIT → **Apache-2.0** (grant de brevet, vital vu la crypto PQC).
- **Dépendances externes (les deux seules du projet)** : `filippo.io/mldsa` (FIPS 204,
  ML-DSA-65), en attendant `crypto/mldsa` dans la stdlib Go (interne depuis 1.26, public
  proposé pour 1.27 — golang/go#77626) ; et `golang.org/x/crypto/argon2` (KDF du vault).
  Toutes les autres primitives restent stdlib Go pure.

### Sécurité
- **Signatures post-quantiques (durcissement Shor)** : les messages sont désormais signés
  par **Ed25519 ET ML-DSA-65** sur le même transcript (`crypto.go`) — usurper un
  expéditeur exige de casser les deux schémas, symétriquement au KEM hybride existant
  (X25519 ⊕ ML-KEM-768). L'allowlist réseau (`serve --authz`, `authz.go`) vérifie
  désormais aussi les deux signatures avant d'autoriser un expéditeur. Le certificat
  TLS auto-signé du transport (`tlsbus.go`) reste Ed25519 classique — `crypto/tls`/`x509`
  n'acceptent pas de certificat feuille ML-DSA en Go 1.25 (voir `SECURITY.md`).
- **Vault durci en Argon2id** : PBKDF2-SHA256 (600 000 itérations, coût CPU seul) remplacé
  par Argon2id (RFC 9106 §7.3 : time=1, mémoire=64 Mio, 4 threads) — résistant aux
  attaques par matériel dédié (GPU/ASIC), pas seulement au brute-force CPU.
- **`recovery combine`** : le secret Shamir reconstitué est désormais protégé par un checksum
  SHA-256 tronqué (4 octets, embarqué par `recovery split` avant découpage). Trouvé par audit
  (2026-07-03) : sous le seuil K, l'interpolation de Lagrange renvoie une valeur bien formée mais
  **fausse** (propriété documentée du schéma Shamir) — et comme n'importe quelle graine de 32
  octets dérive une identité « valide » en apparence (`deriveIdentity`), `combine` acceptait
  silencieusement une reconstruction invalide et **écrasait le vault sans confirmation**. Le
  checksum rejette maintenant toute reconstruction incorrecte avant dérivation.
- **`recovery combine` / `recovery from-phrase`** : écraser un `identity.vault` **déjà existant**
  exige désormais `--force` (le fingerprint reconstruit est affiché avant le refus, pour
  vérification manuelle). `recovery from-phrase` s'appuyait déjà sur le checksum BIP-39 natif pour
  la validité de la phrase, mais n'avait pas non plus de garde-fou contre l'écrasement silencieux.
- **`cmdInbox` / `cmdRemote`** : le repli en clair (`maybeSeal` échoue faute de contact ou de
  vault déverrouillable) ne se signalait qu'en **creux** — l'absence de la mention « chiffré E2E »
  était le seul indice d'un envoi non scellé. `encryptionLabel()` rend maintenant le repli
  **explicite** : « EN CLAIR (aucun contact chiffré connu) ».
- **`remote --tls`** sans `--pin` vers une cible **non-loopback** acceptait silencieusement
  n'importe quel certificat serveur. Avertissement visible désormais (`shouldWarnUnpinnedTLS`),
  symétrique de celui déjà émis côté `serve` sans `--authz`.
- Commentaire de tête de `net.go` corrigé : il annonçait le TLS hybride PQC comme « phase
  suivante » alors qu'il est **déjà livré** (`tlsbus.go`) ; ce qui manque réellement est
  l'authentification mutuelle au niveau TLS (certificat client).
- **`hookInstallFor`** : un provider inconnu (ex. `communikey hook --install opencode`)
  retombait silencieusement sur le snippet de câblage **Claude**, sans le signaler. Avertit
  désormais explicitement et renvoie vers `communikey provider list`.

### Tests
- `relations_test.go` : le graphe familial (`link`/`unlink`/`childrenOf`/`parentOf`/`wouldCycle`,
  roundtrip JSON) n'avait aucun test automatisé — angle mort confirmé et comblé (9 tests, aucun
  bug latent trouvé).

### CI / supply-chain
- GitHub Actions épinglées par **SHA complet** (`actions/checkout`, `actions/setup-go`,
  `anchore/sbom-action`, `sigstore/cosign-installer`, `goreleaser/goreleaser-action`,
  `actions/attest-build-provenance`) au lieu d'un tag majeur mutable, avec commentaire `# vX`
  pour la lisibilité.

Voir [`docs/NEXT.md`](docs/NEXT.md) pour le reste (provenance SLSA/cosign — en attente de la
facturation GitHub —, passkey WebAuthn, surface MCP, audit cryptographique externe). **Go 1.24
est officiellement en fin de vie depuis le 2026-02-10** (politique 2-releases de go.dev, vérifié
2026-07-05) : un bump vers 1.25/1.26 est recommandé mais non fait ici — nécessite un accès réseau
à go.dev/dl.google.com absent du bac à sable de développement, et une mise à jour du toolchain
global hors autorisation explicite.

## [0.2.0] — 2026-06-27

Première base documentée — **alpha**. csend passe du simple injecteur inter-sessions
cmux à un **bus de messages universel** : coopératif, cross-OS/provider, chiffré de bout
en bout et résistant au quantique, avec mémoire persistante et recovery. Tout l'ensemble
ci-dessous est dans l'arbre et couvert par la suite de tests (`go test -race ./...`
vert) ; rien n'est promis sans être livré.

### Ajouté

- **Bus coopératif (colonne vertébrale universelle).** Inbox par fichiers, durable et
  sans démon : `register`, `inbox`, `recv`, `agents`, `whoami`. Marche sur tout OS et
  tout provider, là où l'injection clavier ne peut pas aller.
- **Injection live state-aware (Unix).** Backends **cmux** et **tmux** interchangeables
  derrière une interface `Backend` ; lecture de l'état (`● idle` / `◐ busy` /
  `⚠ confirm`) **avant** d'écrire ; modes de livraison gouvernés (`--stage`, `--send`,
  `--force`). Détection d'état **Claude** calibrée sur de vrais écrans.
- **Graphe familial.** `tree`, `link`/`unlink`, broadcasts `--up`/`--down`/
  `--to-siblings`/`--to-descendants`, avec garde anti-cycle.
- **Chiffrement E2E hybride post-quantique.** Messages signés **Ed25519**, chiffrés
  **AES-256-GCM** sous une clé dérivée de **X25519 ⊕ ML-KEM-768** (HKDF-SHA256).
  Scellement automatique sur le fil dès qu'un contact est connu (`contact add`/`list`,
  jetons `csend id --export`).
- **Identités & vault.** `id --create`/`--export` ; identité hybride dérivée d'une graine
  maître de 32 octets ; vault au repos scellé **AES-256-GCM** avec clé **PBKDF2-SHA256**
  (600 000 itérations). Passphrase via `CSEND_VAULT_PASS_FILE` (recommandé) ou
  `CSEND_VAULT_PASS`.
- **Recovery.** Partage par seuil **Shamir N-sur-M** sur GF(2⁸) (`recovery split`/
  `combine`) et **phrase BIP-39** de 24 mots (`recovery phrase`/`from-phrase`), wordlist
  anglaise officielle embarquée.
- **Réseau machine-à-machine.** `serve`/`remote` (frame JSON) avec payload chiffrable
  E2E, et **TLS 1.3 hybride PQC** (`X25519MLKEM768`) via cert self-signed Ed25519 +
  épinglage d'empreinte (`--tls`, `--pin`). Cible : loopback/LAN.
- **Mémoire.** Journal append-only interrogeable (hash du contenu, jamais le clair) +
  registre persistant des sessions.
- **Aide accessible** sous toutes les formes (`h`, `-h`, `--h`, `help`, `-help`,
  `--help`, `-?`).
- **Outillage de publication.** CI GitHub Actions (`go vet` + `go test -race` + build),
  `Makefile`, `install.sh` (binaire universel arm64 + x86_64), configuration GoReleaser
  (binaires multi-plateformes + formule Homebrew), `PROJECT.nfo`, site statique.

### Corrigé

- **Anti-auto-injection.** `selfRef()` comparait un UUID (`CMUX_SURFACE_ID`) à des cibles
  en forme `surface:N`, si bien que le garde « ne pas s'écrire à soi-même » ne matchait
  jamais. Source d'identité faisant désormais autorité (`cmux tree` / `here=true`), avec
  test de non-régression.

### Sécurité

- Toutes les primitives proviennent de la **stdlib Go 1.24** — **zéro dépendance
  externe**, aucune crypto maison.
- L'assemblage cryptographique **n'a pas encore été audité en externe** ; voir
  [`SECURITY.md`](SECURITY.md) pour le modèle de menace et les limites connues (PBKDF2 vs
  Argon2id, auth réseau, anti-rejeu).

### Limites connues

- Injection clavier **Unix uniquement** (Windows natif et mobile = coopératif-only).
- Adaptateurs Codex/Gemini, passkey, auth mutuelle réseau, clients mobiles et surface
  MCP : **non livrés** (feuille de route).

[Non publié]: https://github.com/aissablk1/communikey/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/aissablk1/communikey/releases/tag/v0.2.0
