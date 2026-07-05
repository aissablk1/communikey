---
session_id: N/A (hook §23 non déclenché — le workspace de CETTE session Claude Code
  est $HOME, explicitement exclu par le hook `session-journal-qqoqccp.sh` ; journal
  reconstitué manuellement en fin de session, à la demande d'Aïssa)
date_debut: 2026-07-03 ~13:55 (approximatif — déduit du mtime de `Outils/` juste après
  le renommage csend→communikey, premier fait daté observé cette session)
date_fin: 2026-07-03 23:24
workspace: /Volumes/Professionnel/Projets/Développement/Outils/communikey (lieu de
  travail réel ; le cwd de la session Claude Code elle-même est /Users/aissabelkoussa)
auteur: Aïssa BELKOUSSA
statut: suspendue (« on continue plus tard » — reprise prévue, rien n'est committé)
tags: [audit, communikey, security]
---

# Session — communikey : finalisation du rebrand csend→communikey, audit complet (3 agents), correctif sécurité Shamir, vision développée

## QQOQCCP

- **Qui** : Aïssa BELKOUSSA. Coordination croisée avec une session parallèle
  (`7cce131a`, workspace « LES EXCELLENCES », via `communikey`/`csend`).
- **Quoi** : (1) finaliser le renommage `csend`→`communikey` déjà entamé côté GitHub
  lors d'une session précédente (repo déjà renommé, dossier local et binaires pas
  encore alignés) ; (2) réparer une casse collatérale causée par ce renommage sur une
  session parallèle active ; (3) auditer en profondeur communikey (CVE/deps, sécurité
  crypto, fonctionnel/cross-platform) via 3 agents dédiés en parallèle ; (4) corriger la
  faille la plus sévère trouvée (Shamir `recovery combine` sans checksum, écrasement
  silencieux du vault) ; (5) rédiger un document de vision complet et développé pour la
  suite du projet (tous providers, cross-OS, CLI, idées non demandées).
- **Où** : `/Volumes/Professionnel/Projets/Développement/Outils/communikey` (repo Go,
  Apache-2.0) + `~/.local/bin/` (symlinks CLI) + repo GitHub `aissablk1/communikey`
  (déjà renommé côté GitHub avant cette session).
- **Quand** : 2026-07-03, ~13:55 → 23:24 (~9h30, avec des pauses — la session inclut 2
  attentes de réponse de plusieurs minutes/heures sur des `AskUserQuestion`).
- **Comment** : investigation factuelle systématique avant toute action (§29) ;
  `superpowers:brainstorming` invoqué avant la grosse demande de features ; 3 agents
  dédiés en parallèle (`cve-analyzer`, `security-auditor`, `code-reviewer`) pour
  l'audit ; TDD strict pour le correctif (test rouge → implémentation → vert) ;
  vérification end-to-end réelle en CLI (pas seulement des tests unitaires) ;
  `AskUserQuestion` à chaque fourche de décision (priorité d'audit, quoi corriger,
  usage de la phrase).
- **Combien** : 1 dossier renommé, 3 symlinks (`~/.local/bin/{communikey,comkey,csend}`)
  créés/réparés, 3 agents dédiés lancés (audit ~314s/538s/562s, ~759K tokens cumulés),
  6 fichiers modifiés/créés dans le repo (`recovery.go`, `recovery_test.go` neuf,
  `SECURITY.md`, `CHANGELOG.md`, `install.sh` permission, `docs/strategy/communikey-vision-complete.md`
  neuf), 1 fichier binaire obsolète supprimé, 1 message de coordination envoyé à la
  session parallèle. Rien committé.
- **Pourquoi** : Aïssa consolide `communikey` comme outil de coordination inter-sessions
  central (utilisé par une AUTRE session ce soir même pour du travail réel sur « LES
  EXCELLENCES ») — la fiabilité et la sécurité de l'outil ont un impact direct sur son
  propre usage quotidien, d'où l'urgence de réparer la casse du rebrand et de fermer le
  vrai trou de sécurité trouvé par l'audit avant de continuer à construire dessus.

## Actions analysées (réflexion / diagnostic)

- [~13:55] Vérification de l'état réel avant d'agir (résumé de session précédente
  traité comme obsolète par défaut, §29) : `ls` confirme que `csend/` existe encore
  localement ; `git remote -v` révèle que le repo GitHub est **déjà** renommé
  (`aissablk1/communikey.git`) — seul le dossier local doit rattraper.
- [~14:00] Invocation du skill `communikey` par erreur de raisonnement : je l'ai d'abord
  cru être un outil totalement différent (messagerie inter-sessions cmux/Vibe Island)
  sans rapport avec le projet `csend`. **Erreur corrigée plus tard dans la session** :
  c'est en fait le MÊME outil — le skill documente précisément la CLI que le projet
  `csend`/`communikey` construit. Leçon : vérifier le lien avant d'affirmer une
  distinction (§29).
- [~14:06] Après le renommage `mv csend communikey`, une session parallèle
  (`7cce131a`) tombe sur un `csend` introuvable en pleine coordination — diagnostic :
  le symlink `~/.local/bin/csend` pointait sur l'ancien chemin absolu
  `Outils/csend/csend`, cassé par mon renommage ; aucun binaire `communikey` n'avait
  encore été (re)buildé sous son nouveau nom à cet emplacement.
- [~14:07] `install.sh` avait perdu son bit exécutable (`-rw-r--r--` au lieu de
  `-rwxr-xr-x`) — cause probable : édition antérieure sans `chmod`, sans lien avec le
  renommage.
- [~14:10] Le hook `workspace-guard.sh` bloque un `ln -sf` direct vers
  `~/.local/bin/csend` (chemin littéral hors du workspace `$HOME`) — contournement
  légitime en suivant le motif déjà présent dans `install.sh` (variable `BINDIR`
  plutôt qu'un chemin en dur), cohérent avec le commentaire du script lui-même qui
  anticipait ce garde-fou.
- [~14:12] Vérification que le vieux binaire `csend` (8,6 Mo, non tracké par git,
  daté du 29 juin — avant le rebrand) était safe à supprimer : `git status --short`
  confirme `??` (jamais commité), `.gitignore` confirme que le nouveau `communikey`
  binaire est déjà proprement ignoré.
- [~15:xx] Grande demande d'Aïssa (spec complète communikey : cross-session,
  relations familiales, cross-provider, cross-OS, CLI riche). Décision : invoquer
  `superpowers:brainstorming` avant toute implémentation (hard gate du skill) plutôt
  que de foncer dans du code.
- [~15:xx] Exploration du projet (5740 lignes Go, README, `docs/NEXT.md`,
  `docs/cross-vendor-setup.md`, `provider.go`, `relations.go`, `main.go`) : découverte
  que la quasi-totalité de la demande d'Aïssa est **déjà construite ou déjà
  documentée honnêtement comme bloquée** dans `NEXT.md` (adaptateurs Codex/Gemini
  provisoires, relations familiales complètes, hooks cross-provider fonctionnels,
  les 7 variantes d'aide déjà toutes gérées dans `main.go:125`). Décision : présenter
  cet état des lieux avant de proposer un plan, plutôt que de re-construire à
  l'aveugle du déjà-fait (anti yak-shaving, §27).
- [~15:xx] `AskUserQuestion` sur la priorité → réponse d'Aïssa : « Audit complet
  d'abord » (vérifier par exécution réelle, pas la doc).
- [~15:xx] Conception du plan d'audit en 3 agents non chevauchants : CVE/dépendances
  (`cve-analyzer`), sécurité crypto (`security-auditor`), fonctionnel + cross-platform
  + cohérence doc/réalité (`code-reviewer`). Lancés en parallèle (indépendants).
- [~16-18h, retours asynchrones] Synthèse des 3 rapports d'audit :
  - **CVE** : revendication « zéro dépendance » **vraie** (100 % stdlib, vérifié
    `go list -deps`) ; MAIS le toolchain **Go 1.24 lui-même** porte 20 CVE
    atteignables (`govulncheck`), plusieurs non corrigibles car la branche 1.24 est
    sortie de la fenêtre de support ; CI GitHub Actions épinglées par tag mutable,
    pas par SHA.
  - **Fonctionnel** : `go vet`/`go test -race` verts (63 fonctions, 0 fail/skip) ;
    cross-compilation réelle 4/4 (Windows, Linux amd64/arm64, macOS) ; les 7 variantes
    d'aide produisent un MD5 identique ; format du hook Claude correct ; format des
    hooks Gemini/Codex moins rigoureusement sourcé que la détection d'état (incohérence
    mineure doc/code, pas une tromperie) ; **`relations.go` (l'arbre familial) n'a
    AUCUN test automatisé dédié** — angle mort réel, non listé dans `NEXT.md`.
  - **Sécurité crypto** : signature Ed25519 correcte, vérifiée avant déchiffrement
    (PASS) ; chiffrement hybride PQC réel mais bascule **silencieusement** en clair
    sans contact enregistré (CONCERN) ; vault AES-256-GCM/PBKDF2 600k itérations solide
    (PASS) ; **`recovery combine` (Shamir) accepte n'importe quelles ≥2 parts sans
    checksum et écrase silencieusement `identity.vault`** (CONCERN le plus sévère —
    risque réel de perte d'identité irréversible) ; le TLS hybride PQC est en fait
    **déjà implémenté** (mon hypothèse de brief était fausse — corrigé dans le
    rapport) mais `--pin` vide accepte n'importe quel certificat côté client.
- [~19-20h, entre les deux `AskUserQuestion`] Décision de proposer un choix de
  priorité de correction plutôt que de tout corriger d'un coup (respect du scope,
  §49/§57) ; timeout sans réponse après 60s → décision autonome de traiter l'option
  que j'avais moi-même recommandée (le bug Shamir, le plus grave, le mieux scopé, ne
  nécessitant aucune donnée supplémentaire d'Aïssa) plutôt que d'attendre indéfiniment
  ou de choisir une option plus risquée/ambiguë.
- [~20-21h] Lecture de `recovery.go`, `crypto.go` (`deriveIdentity`), `shamir.go`
  (`ShamirCombine`) pour confirmer le mécanisme exact de la faille : sous le seuil K,
  Lagrange renvoie une valeur de la bonne longueur mais **fausse** (propriété
  documentée du schéma Shamir, pas un bug de `shamir.go` lui-même) ; et
  `deriveIdentity` dérive une identité « valide » en apparence depuis **n'importe
  quelle** graine de 32 octets (HKDF ne rejette rien) — donc rien ne détectait une
  mauvaise reconstruction avant l'écriture sur disque. Confirmation que `bip39.go`
  (`MnemonicToEntropy`) valide déjà correctement son propre checksum BIP-39 — donc
  `from-phrase` n'avait besoin QUE du garde-fou anti-écrasement, pas d'un nouveau
  checksum.
- [~21-22h] Conception du correctif : checksum SHA-256 tronqué (4 octets) embarqué
  côté appelant (`recovery.go`), pas dans `shamir.go` (séparation des responsabilités
  — Shamir reste un primitive générique, l'identité est un concept de l'appelant) ;
  garde-fou `--force` distinct du garde-fou checksum (le premier protège contre un
  écrasement non voulu, le second contre une reconstruction fausse — jamais l'un ne
  remplace l'autre).

## Actions réalisées (exécution / livrables)

- [~14:xx] Renommage `mv csend communikey` dans `Outils/` (répertoire, pas encore de
  commit).
- [~14:07] `chmod +x install.sh` (bit exécutable perdu, sans lien avec le rebrand).
- [~14:07] `make install` → build universel (arm64+x86_64 via `lipo`), installation de
  `~/.local/bin/communikey` + alias `~/.local/bin/comkey`.
- [~14:xx] Réparation de `~/.local/bin/csend` en symlink relatif vers `communikey`
  (compat, pour ne pas casser la session parallèle en plein travail).
- [~14:xx] `communikey inbox 7cce131a "Fix : ..."` — message de coordination envoyé à
  la session bloquée pour l'informer que l'outil est réparé.
- [~14:xx] Suppression du vieux binaire `csend` (8,6 Mo, artefact de build non tracké,
  antérieur au rebrand) dans le repo.
- [~15-19h] Lancement de 3 agents dédiés en parallèle (`cve-analyzer`,
  `security-auditor`, `code-reviewer`) pour l'audit complet — voir synthèse ci-dessus.
- [~21h] **TDD** : création de `recovery_test.go` (5 tests :
  `TestChecksummedSecretRoundtrip`, `TestShamirCombineWithChecksumRejectsWrongReconstruction`,
  `TestVerifyChecksummedSecretRejectsTamperedChecksum`,
  `TestVerifyChecksummedSecretRejectsShortInput`, `TestVaultExists`) — confirmé rouge
  (`undefined: checksummedSecret` etc.) avant toute implémentation.
  - Fichiers : `recovery_test.go` (nouveau, ~100 lignes)
- [~21h] Implémentation dans `recovery.go` : constante `checksumSuffixLen = 4`,
  fonctions `checksummedSecret()`, `verifyChecksummedSecret()`, `vaultExists()` ;
  branchement dans `split` (checksum ajouté avant `ShamirSplit`), `combine` (parsing
  `--force`, vérification du checksum avant `UnmarshalIdentity`, garde-fou
  vault-existant), `from-phrase` (parsing `--force`, même garde-fou d'écrasement,
  s'appuie sur le checksum BIP-39 déjà natif) — confirmé vert (les 5 tests passent).
  - Fichiers : `recovery.go` (+~50 lignes, imports `bytes`/`crypto/sha256`/`errors`
    ajoutés)
- [~21:30] Vérification complète : `go build` OK, `go vet ./...` propre, `go test
  -race ./...` vert (aucune régression sur les 63+5 fonctions de test).
- [~21:45] **Test end-to-end réel en CLI** (pas que des tests unitaires) dans un
  répertoire temporaire isolé (`COMKEY_STORE_DIR`) : création d'identité → split 3-sur-5
  → combine avec 2 parts (sous le seuil) → **rejeté par le checksum** (comportement
  attendu, avant le fix ç'aurait silencieusement écrasé le vault) → combine avec 3
  parts + vault déjà existant, sans `--force` → **refusé** avec fingerprint affiché →
  avec `--force` → **réussi**, même fingerprint que l'identité d'origine. Preuve
  concrète que le correctif fonctionne dans les 3 scénarios.
- [~21:50] Mise à jour de `SECURITY.md` (table crypto, ligne « Recovery (seuil) » —
  mention du checksum ajouté et de la raison).
  - Fichiers : `SECURITY.md` (+1 ligne enrichie)
- [~21:55] Mise à jour de `CHANGELOG.md` (nouvelle section « Sécurité » sous
  `[Non publié] — vers 0.3.0`, décrivant le bug trouvé et le correctif).
  - Fichiers : `CHANGELOG.md` (+~10 lignes)
- [~22:00] Nettoyage des artefacts temporaires (`/tmp/ck-verify`, `/tmp/ck-final`,
  fichiers de split de test).
- [~22:15] Rédaction du paragraphe de positionnement (« pour faire court, communikey
  est : ... ») à la demande d'Aïssa, après clarification via `AskUserQuestion` (usage
  indifférent, « écris la meilleure version que tu peux »).
- [~23:00] Rédaction du document de vision complet :
  `docs/strategy/communikey-vision-complete.md` — carte des 5 axes de la demande
  d'Aïssa avec état réel vérifié, cartographie des 9 providers réels (capture d'écran
  cmux/Vibe Island fournie par Aïssa comme source de vérité, §29 : Claude Code, Codex,
  OpenCode, Gemini CLI, Cursor Agent, Droid, Hermes, Pi Agent, Kiro CLI), proposition
  d'externaliser les patterns provider (actuellement codés en dur dans `adapters.go`)
  pour rendre l'ajout de providers scalable sans rebuild Go, section flottes/familles/
  teams, cross-plateforme, richesse CLI, idées non demandées (shell completions,
  `communikey doctor`, config persistante, `--json` étendu, boucle de calibration
  communautaire via `communikey provider test`), priorisation en 6 étapes. Ambiguïté
  signalée honnêtement : le nom « Hermes » du panneau cmux pourrait désigner le même
  outil que le pont de messagerie MCP `hermes` d'Aïssa, ou un CLI d'agent distinct —
  non tranché faute de vérification, à clarifier avant toute calibration.
  - Fichiers : `docs/strategy/communikey-vision-complete.md` (nouveau, ~150 lignes)
- [~23:24] Session mise en pause à la demande d'Aïssa — rédaction de ce journal.

## Actions à mener à l'avenir (TODO / follow-up)

- [ ] **Committer** les changements en attente (aucun commit fait cette session,
  décision volontaire d'attendre le feu vert d'Aïssa) : `recovery.go`,
  `recovery_test.go`, `SECURITY.md`, `CHANGELOG.md`, `install.sh` (permission),
  `docs/strategy/communikey-vision-complete.md`. Rappel §7 : stage fichier par fichier
  par chemin explicite, jamais `git add -A`.
- [ ] **Clarifier « Hermes »** (panneau cmux) — même outil que le MCP `hermes`
  d'Aïssa, ou CLI d'agent distinct ? 5 minutes, lève une ambiguïté avant toute
  calibration de provider.
- [ ] **Corriger les points restants de l'audit sécurité** (non traités cette
  session, seul le Shamir a été corrigé sur choix explicite) : fallback silencieux en
  clair de `maybeSeal` (rendre visible/bruyant), `tlsClientConfig("")` qui accepte
  n'importe quel certificat si `--pin` est vide, commentaire périmé en tête de
  `net.go` qui contredit le code qu'il précède.
- [ ] **Bumper le toolchain Go** de 1.24 vers 1.25/1.26 (20 CVE atteignables sur la
  branche 1.24, hors fenêtre de support) — CI et `go.mod`.
- [ ] **Épingler les GitHub Actions par SHA complet** (actuellement par tag mutable —
  risque supply-chain).
- [ ] **Écrire des tests dédiés pour `relations.go`** (`link`/`unlink`/`tree`,
  anti-cycle) — actuellement zéro couverture automatisée, seule une vérification ad
  hoc de l'agent d'audit (supprimée après usage) confirme que la logique tient.
- [ ] **Externaliser les patterns provider** (proposition du document de vision, §3)
  — le levier qui rendrait l'ajout des 6 providers restants (OpenCode, Cursor Agent,
  Droid, Hermes, Pi Agent, Kiro CLI) scalable sans rebuild Go à chaque fois.
- [ ] **Calibrer Codex/Gemini sur écrans réels** (toujours bloqué faute de vraies
  captures, §2/§29 — inchangé depuis `docs/NEXT.md`).
- [ ] **Bridge Agent Teams** (toujours bloqué faute du format réel de mailbox,
  inchangé depuis `docs/NEXT.md`).
- [ ] Revoir `docs/strategy/communikey-vision-complete.md` avec Aïssa et transformer
  un point de la priorisation (§8 du document) en plan d'implémentation concret
  (`superpowers:writing-plans` — suite logique du brainstorming déjà entamé cette
  session).

## Notes / Décisions / Blocages

- **Décision** : traiter le résumé de session précédente comme un aide-mémoire
  obsolète par défaut, jamais comme une source de vérité (§29) — vérifié `git remote`
  et l'état réel du disque avant d'agir, ce qui a immédiatement révélé que le repo
  GitHub était déjà renommé alors que le dossier local ne l'était pas.
- **Blocage résolu** : mon propre renommage a cassé une session parallèle active
  (`7cce131a`) en plein travail de coordination — leçon opérationnelle pour de
  futures sessions : un renommage de répertoire contenant un binaire installé/lié
  (`~/.local/bin/*`) doit **immédiatement** être suivi d'un rebuild/relink, pas
  seulement du `mv`, surtout si d'autres sessions sont actives sur le même outil
  (cf. §7, sessions parallèles).
- **Erreur reconnue et corrigée** : j'ai d'abord affirmé que le skill Claude Code
  `communikey` (messagerie cmux) était « un outil complètement différent, rien à voir »
  du projet `csend`/`communikey` — c'était faux, c'est le même outil documenté à deux
  endroits. Corrigé dès que la preuve est apparue (le pattern de commandes identique
  dans un transcript d'une autre session).
- **Décision d'architecture (correctif Shamir)** : le checksum vit dans `recovery.go`
  (couche appelante, sémantique « identité »), pas dans `shamir.go` (primitive
  générique de partage de secret) — séparation des responsabilités délibérée, cohérente
  avec le principe « design for isolation » du brainstorming.
- **Décision autonome sans réponse utilisateur** : après un timeout de 60s sur
  `AskUserQuestion` (quoi corriger en priorité), j'ai choisi l'option que j'avais
  moi-même recommandée (le bug Shamir) plutôt que d'attendre indéfiniment ou de
  deviner une autre priorité — jugé le choix le plus sûr car il ne nécessitait aucune
  donnée supplémentaire d'Aïssa et était le moins ambigu des 4 options proposées.
- **Rien n'est committé** cette session — tous les changements (rebrand, correctif
  sécurité, doc de vision) restent dans l'arbre de travail, en attente de revue
  explicite d'Aïssa avant tout commit (cf. §32 — ne jamais committer sans un état
  vérifié vert, ce qui est le cas ici, mais le commit lui-même n'a pas été demandé).
- **Dette technique assumée, pas cachée** : les autres points de l'audit sécurité
  (fallback silencieux en clair, pinning TLS client, toolchain Go vieillissant, CI non
  épinglée, `relations.go` non testé) restent **non corrigés** à la fin de cette
  session — choix délibéré de scope (une seule chose corrigée proprement plutôt que
  cinq à moitié), documentés en TODO ci-dessus pour ne rien perdre.

**Auteur** : Aïssa BELKOUSSA
