# communikey — session 2026-07-06 : signature hybride ML-DSA + vault Argon2id

## Qui / Quoi / Où / Quand / Comment / Combien / Pourquoi

- **Qui** : Aïssa BELKOUSSA, via une session Claude Code.
- **Quoi** : (1) durcissement post-quantique de la signature applicative
  (Ed25519 → hybride Ed25519 ⊕ ML-DSA-65) suite à une demande initiale
  incohérente (liste de 17 « sécurités » mêlant primitives légitimes,
  algorithmes obsolètes/cassés — RSA-1024, MD5, SHA-1 —, et objets
  non-cryptographiques — Shor, GNFS, Coppersmith–Winograd, Busy Beaver,
  classification des groupes finis simples) ; (2) après clarification et
  QCM, mise en œuvre ciblée sur le seul item légitime et déjà sur la
  roadmap : ML-DSA. (3) Sur demande explicite d'Aïssa, upgrade du KDF du
  vault PBKDF2 → Argon2id.
- **Où** : `/Volumes/Professionnel/Projets/Développement/Outils/communikey`
  (repo Git réel, `main`).
- **Quand** : 2026-07-06.
- **Comment** : audit du repo réel avant toute implémentation (`crypto.go`,
  `SECURITY.md` déjà en place) ; vérification de l'API exacte via `go doc`
  après ajout des dépendances (`filippo.io/mldsa`, `golang.org/x/crypto/argon2`)
  plutôt que de coder de mémoire ; tests ciblés ; documentation mise à jour
  partout où l'ancien état était cité ; commits atomiques séparés.
- **Combien** : 2 commits, 14 fichiers touchés au total, suite de tests
  complète verte à chaque étape (build + vet + test + mod verify).
- **Pourquoi** : durcissement post-quantique cohérent avec l'architecture
  hybride déjà en place (KEM X25519 ⊕ ML-KEM-768) ; résistance accrue du
  vault au brute-force matériel (GPU/ASIC) que PBKDF2 ne couvre pas.

## Actions réalisées

1. **[commit `24c4227`]** `feat(crypto): signature hybride Ed25519 ⊕ ML-DSA-65`
   — `Identity`/`PublicBundle`/`SealedMessage` gagnent le volet ML-DSA-65 ;
   `Seal`/`Open` exigent les DEUX signatures ; `authz.go` (`serve --authz`)
   mis à jour en cohérence (vérifiait auparavant Ed25519 seule — trouvé et
   corrigé dans le même geste). Dépendance `filippo.io/mldsa` (API vérifiée
   via `go doc`, pas de mémoire) — pont explicite vers le futur
   `crypto/mldsa` stdlib (Go 1.27, golang/go#77626). `go.mod` : `go 1.24` →
   `go 1.25` (exigé par la dépendance ; auto-toolchain a résolu go1.25.11
   sans intervention). 5 tests ajoutés (falsification ML-DSA seule /
   Ed25519 seule / substitution de clé). CI/release alignés (`go-version`).
   Doc : `SECURITY.md`, `THREAT-MODEL.md`, `README.md`, vision stratégique.
2. **[commit `eb73942`]** `security(vault): Argon2id (RFC 9106) au lieu de
   PBKDF2` — sur décision explicite d'Aïssa (accepter la 2ᵉ dépendance
   externe, point qui était « à arbitrer »). Paramètres RFC 9106 §7.3
   (time=1, mémoire=64 Mio, 4 threads) cités depuis la doc du paquet
   `golang.org/x/crypto/argon2` (vérifiés, pas de mémoire). BREAKING assumé
   (alpha, zéro utilisateur externe) : un vault PBKDF2 antérieur ne se
   déverrouille plus. Doc mise à jour partout où PBKDF2 était cité
   (`SECURITY.md`, `THREAT-MODEL.md`, `README.md`, `docs/NEXT.md`, vision).

## Ce qui a été refusé / réorienté (honnêteté §29)

La demande initiale listait 17 éléments comme « sécurités » à ajouter.
Beaucoup n'étaient pas des primitives de sécurité valides : **Shor** et
**GNFS** sont des algorithmes d'*attaque* (contre lesquels le KEM/la
signature hybrides existent déjà), **Coppersmith–Winograd** est un résultat
de complexité de multiplication matricielle sans rapport, **Busy Beaver**
est une fonction de calculabilité non calculable en général, la
**classification des groupes finis simples** est un théorème de
mathématiques pures. **RSA-1024, MD5, SHA-1** auraient été des régressions
(cassés/dépréciés). **FairPlay** (DRM Apple) et **UPX+AES** (« chiffrement
de firmware ») étaient hors-sujet. Rien de tout cela n'a été implémenté —
documenté et expliqué à Aïssa avant toute action, avec verdict par item.

## Blocage constaté (non contourné)

- **Calibrage Gemini** : toujours bloqué, **re-vérifié ce jour** —
  `gemini` CLI 0.40.1 installé, mais `~/.gemini/` ne contient aucun fichier
  de credentials OAuth et `GEMINI_API_KEY` est absente de l'environnement.
  Ce blocage nécessite une action directe d'Aïssa (fournir une clé API, ou
  lancer `gemini` lui-même pour compléter le flux OAuth interactif —
  impossible à faire headless pour l'agent).

## Session parallèle constatée

Au moment de committer, le registre `~/.claude/sessions-active/` a montré
une autre session active sur ce même repo, éditant
`docs/superpowers/specs/2026-07-06-communikey-model-provider-design.md`
(conception providers). Signalé à Aïssa avant tout commit — confirmé être
lui-même en parallèle, aucune collision réelle. Staging fait par chemin
explicite dans les deux commits (jamais `git add -A`), ce fichier jamais
touché.

**Auteur** : Aïssa BELKOUSSA
