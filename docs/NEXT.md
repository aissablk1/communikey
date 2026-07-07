# communikey — ce qui reste, et pourquoi (honnêteté §29)

État au 2026-06-27. Ce fichier liste ce qui n'est **pas** encore livré, avec la **raison
réelle** (souvent un blocage physique : besoin d'un appareil, d'un autre runtime, ou de
vraies données à observer) et le **plan**. Rien ici n'est fait à moitié et présenté comme
fini.

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

- ~~Réseau multi-machine durci (TLS hybride PQC + auth mutuelle).~~ **Livré 2026-07-07** :
  `serve --tls` (TLS 1.3 hybride `X25519MLKEM768`) + `serve --tls --authz` (auth MUTUELLE —
  certificat client dérivé de l'identité, vérifié contre l'allowlist au handshake,
  `serverTLSConfigMutual` dans `tlsbus.go`). Reste ouvert : le durcissement au-delà de
  cette auth mutuelle pour une exposition Internet large (rate-limiting réseau, etc.).

- **Clients mobiles (iOS / iPadOS / Android).** Runtime distinct (app/PWA). Rappel honnête :
  le mobile ne peut **pas** injecter au clavier (sandbox) → il rejoint le bus comme **client**
  (envoyer / relever / approuver / être notifié, façon omnara). *Plan* : client mince vers le
  réseau communikey (`serve`).

- **Windows natif (injection clavier).** Physiquement impossible : pas de PTY maître
  partageable, ConPTY ne s'attache pas à un process déjà lancé. Windows reste **coop-only**
  (inbox). C'est une limite assumée, pas un manque.

## Bloqués par l'absence de vraies données (ne pas inventer, §2/§29)

- **Adaptateurs d'état Codex / Gemini / Antigravity — confirmation par capture live.**
  Les trois adaptateurs sont livrés (`adapters.go`), calibrés sur source primaire
  (bundle JS, dépôt officiel, ou extraction `strings` du binaire installé selon le
  CLI), mais **aucun n'a encore été confirmé sur un vrai écran capturé en direct**
  (comme `testdata/claude_idle_real.txt`) — Codex absent du PATH, Gemini CLI
  individuel retiré (18/06/2026), Antigravity nécessite une session OAuth Google
  complète pour lancer une conversation réelle. *Plan* : capturer un vrai écran de
  chacun (pseudo-tty, comme `communikey teams` l'a fait pour Agent Teams), lever les
  CAVEATS documentés dans `adapters.go`.

- **Bridge Agent Teams.** La livraison dans la *mailbox* d'Agent Teams exige d'en connaître le
  **format exact** (`~/.claude/teams/…`), non documenté publiquement. *Plan* : inspecter une
  session Agent Teams réelle, puis écrire le writer — sans deviner le format.

## Publication (en attente du feu vert d'Aïssa)

- Repo GitHub public, première Release (GoReleaser), tap Homebrew, déploiement du site.
  Voir `docs/PUBLISHING.md`. Rien n'a été poussé sans accord.

**Auteur** : Aïssa BELKOUSSA
