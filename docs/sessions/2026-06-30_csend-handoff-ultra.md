---
session_id: 6cb99775 (surface:45 — csend-messaging-tool)
date_debut: 2026-06-27
date_fin: 2026-07-02
workspace: /Volumes/Professionnel/Projets/Développement/Outils/csend
auteur: Aïssa BELKOUSSA
statut: TERMINÉE — les 6 tâches faites ; projet rebrandé csend → **Communikey** (repo aissablk1/communikey public, Apache-2.0, CI verte, crypto durcie v0.3.0, provenance SLSA/cosign/SBOM prête, primitif cross-vendor construit+démontré). Ship v0.3.0 DIFFÉRÉ (attente fonds + domaine communikey.dev en cours de réservation) — à un `git tag v0.3.0` près, après le commit du sweep de surface:62.
tags: [communikey, csend, inter-agent, handoff, exploration, strategie, crypto, pqc, rebrand, apache, ci, distribution]
---

# HANDOFF ULTRA — csend (du fix au bus universel, à l'exploration stratégique)

> Document de reprise complet. Écrit sur ordre d'Aïssa pour **vider le contexte sans rien perdre** :
> tout l'état, l'analyse et les tâches vivent ici, sur disque. Une session fraîche doit pouvoir
> reprendre « tout faire » à partir de ce seul fichier.

---

## 0. QQOQCCP

- **Qui** : Aïssa BELKOUSSA (décideur). Exécution = session Claude `surface:45`. Session parallèle
  `surface:62` = distribution/publication (priée de journaliser + se clore le 2026-06-30).
- **Quoi** : analyser puis refondre `csend` (outil d'injection inter-session Claude CLI) en **bus de
  messagerie inter-agents universel** ; le publier ; puis **explorer en profondeur** son espace des
  possibles (16 axes, agent teams) car Aïssa juge l'existant « à 1%, n'apporte rien ».
- **Où** : `…/Outils/csend` (Go). Repo public **github.com/aissablk1/csend** (MIT). Release **v0.2.0**.
- **Quand** : 2026-06-27 → 2026-06-30.
- **Comment** : analyse de code → 2 agents de recherche marché → brainstorm → spec → TDD par phases →
  publication (goreleaser local) → workflow d'exploration 16 agents + synthèse + critique adversariale.
- **Combien** : 42 commits, 49 fichiers Go (21 tests), 33 commandes CLI, suite verte.
- **Pourquoi** : aucun outil du marché ne combine injection-dans-session-externe + state-aware +
  cross-provider + mémoire + crypto. MAIS l'exploration conclut que le vrai créneau défendable est
  étroit (voir §3).

---

## 1. ÉTAT COMPLET DE csend (ce qui existe et marche)

**Définition.** Bus de messagerie **inter-agents** pour CLI, **local**, **Go**, **MIT**, **zéro
dépendance externe** (que la stdlib). Permet à une session d'agent (Claude Code, Codex, Gemini…) d'en
piloter une autre : lire son écran, détecter son état, lui injecter/soumettre un message — ou déposer
dans un inbox coopératif (tout OS).

**Capacités livrées (vérifiées) :**
- **Injection state-aware** dans une session vivante via backends **cmux** (socket JSON-RPC) et **tmux**
  (send-keys/capture-pane). Détection d'état idle/busy/await-confirm/unknown par scraping d'écran.
- **Inbox coopératif fichier** (`inbox`/`recv`/`hook`) : marche partout, sans multiplexeur — c'est le
  backbone universel. L'injection live = repli Unix-multiplexeur.
- **Réception live** : `csend hook` (à câbler en hook `UserPromptSubmit` → les messages arrivent dans
  le contexte de l'agent sans polling) + `csend watch` + `--install`.
- **Crypto E2E hybride post-quantique** : KEM X25519 ⊕ ML-KEM-768 → HKDF → AES-256-GCM, signatures
  Ed25519. Identité dérivée d'**une** graine maître (HKDF domain-séparé).
- **Vault** chiffré (PBKDF2-SHA256 → AES-GCM ; passphrase via `CSEND_VAULT_PASS` ou `…_FILE`).
- **Recovery** : Shamir N-sur-M (GF(256)) + **phrase BIP-39 24 mots** (wordlist officielle embarquée).
- **Réseau** : `serve`/`remote` sur TCP, **TLS 1.3 hybride PQC** (Go 1.24 négocie X25519MLKEM768),
  cert auto-signé Ed25519 + pinning d'empreinte. `serve --authz` (n'accepte qu'un message E2E **signé**
  d'un expéditeur allowlisté). **File offline** (remote met en file si injoignable et rejoue).
- **Anti-replay** (dédup sur le **nonce signé**), **verrou fichier** (anti lost-update sur le registre).
- **Graphe familial** père/enfants + broadcast ; registre + journal interrogeable ; sortie **`--json`**.
- **Providers** : Claude (réel) + adaptateurs Codex/Gemini.
- **Primitive `csend key`** : envoyer une touche brute (enter/escape/ctrl+c…) à une surface.
- **Aide** : `h`/`-h`/`--h`/`help`/`-help`/`--help`/`-?`.
- **CI cross-OS** (Linux/macOS/Windows) + job race + **fuzz** du parser de trame.
- **Service** : unités **launchd** (macOS) + **systemd** (Linux) pour `csend serve` permanent.
- **Démo** : `scripts/demo-two-agents.sh` prouve la boucle (alice écrit → bob reçoit via son hook).

**Distribution (session surface:62).** Repo **public**, **LICENSE MIT**, **SECURITY.md**, **CHANGELOG**,
**release v0.2.0** publiée via **goreleaser local** (a contourné les Actions verrouillées par la
facturation GitHub) avec binaires **toutes plateformes** (darwin/linux amd64+arm64, windows zip) +
`checksums.txt`. Formule **Homebrew** prête (`.goreleaser.yaml`, tap `aissablk1/homebrew-tap` à créer).
Site `site/` (terminal souverain, anti-slop) prêt pour **Cloudflare Pages → csend.dev**.

**Bug corrigé important (commit `b716daf`).** L'envoi `csend send` tapait le texte puis pressait Entrée
**immédiatement** → course de collage → message **tapé mais non soumis** (Claude Code en mode paste
traite l'Entrée comme un saut de ligne), tout en rapportant faussement « submitted ». Fix : **délai
250 ms + VÉRIFICATION après Entrée** (relit l'écran ; si la saisie n'est pas vidée → 2ᵉ tentative ;
sinon « déposé, non validé » honnête). Vérifié : l'envoi à surface:62 se soumet désormais réellement.

---

## 2. DOGFOODING & LEÇONS (incidents de cette session)

- **csend pilote csend** : prouvé en réel — `surface:45` a envoyé des notes de coordination à
  `surface:62` (l'autre session csend) via `csend send`/`csend key`, vérifié par `csend read`.
- **Leçon §29 (faux statut)** : ne jamais croire la sortie d'un outil (« ✓ validé ») sans **vérifier le
  terrain** (relire l'écran). C'est ce qui a révélé le bug de soumission.
- **Leçon §20 (typo)** : j'ai stripé les accents par paresse de quoting bash → faute. Correctif : tout
  message/commit en **français accentué dès le départ** (guillemets doubles bash, l'apostrophe y est
  littérale).
- **Limite réelle (cmux)** : les touches de contrôle d'**effacement** (`ctrl+c`/`escape`/`ctrl+u`) sont
  livrées mais **ne vident pas** le brouillon de Claude Code par injection ; seule l'**Entrée** isolée
  soumet. Le full-edit (clear/iterate) marche sur le backend **tmux**, pas cmux.
- **§7 (sessions parallèles)** : territoires disjoints respectés — `surface:45` = cœur, `surface:62` =
  distribution. Convergence parfaite dans `origin/main`, aucune session n'a écrasé l'autre. `main.go`
  committé sur autorisation explicite d'Aïssa.

---

## 3. L'EXPLORATION DU 99% — VERDICT (le cœur stratégique)

**Méthode.** Workflow « agent teams » : 16 agents explorateurs en parallèle (poussés à la frontière,
avec recherche web), un par axe (portabilité, connectivités, IA/agents, interop/standards, produits,
code/archi, sécurité, crypto, licences, open-source, marque, marketing, site vendeur, business, DX,
perf/scale) → synthèse stratégique → **critique adversariale** défendant le verdict « 1% ». 18 agents,
~4,5 M tokens. (1er run rate-limité → relancé avec **boucle de ré-essai par vagues de 3**, qui a tout
récupéré.) **Doc complet : `docs/exploration/2026-06-29-csend-99pct-exploration.md`.**

**Verdict (la critique gagne, Aïssa a raison).**
- Le « substrat de l'Internet des agents » est une **ambition narrative** : un substrat s'impose par
  l'adoption/lock-in (MCP, A2A), il ne se **décrète** pas en se renommant + en écrivant des adaptateurs.
  csend deviendrait au mieux un **citoyen** du substrat des autres.
- La **crypto** est de la bonne ingénierie **sur le mauvais axe** : deux process même machine/même UID
  n'ont pas besoin de PQC ; la crypto ne vaut qu'**inter-hôtes** = territoire **Tailscale/WireGuard**
  (déjà audité). Donc sur-ingénierée pour localhost, redondante pour le cross-host — **et non auditée**.
- **Erreur de couche (le « moat » M8)** : une « sécurité au niveau du bus » ne s'enforce pas depuis le
  fil ; taguer un champ « ceci est de la donnée » n'empêche pas l'agent de le concaténer dans son
  prompt ; le bus ne voit pas tout le trafic. Le vrai moniteur de référence d'un essaim, c'est **Claude
  Code**, pas csend.
- **Risque mortel (R1)** : doublon avec **Agent Teams natif** (Claude Code), Cursor Composer, Claude
  Squad, ruflo/claude-flow, AWS CAO. La plateforme absorbe le cas d'usage. csend **perd**
  feature-contre-feature.

**Les 3 survivants (vrais, bon marché, défendables — pas des ruptures) :**
1. **M1 — Provenance** (SLSA L3 + cosign keyless + SBOM CycloneDX + builds reproductibles + threat
   model agentique). *Moat-par-diligence* quasi gratuit ; personne ne le fait dans ce coin bricolé.
2. **M3 — Apache-2.0 + marque** (relicence MIT→Apache pour le **grant de brevet** vital vu le PQC ; CLA
   léger avant le 1er PR externe ; dépôt de marque + cert mark « csend-compatible »). **Gratuit
   maintenant tant qu'Aïssa est solo, irréversible plus tard.**
3. **LA niche défendable** : `{cross-vendor + local + chiffré}` — **Claude + Codex + Gemini dans un même
   bus local**. La seule chose qu'Anthropic/Google ne construiront jamais. Marché **minuscule** (power
   users multi-vendor, contextes régulés/air-gapped), déjà contesté par ruflo, mais **structurellement
   imprenable** par un éditeur.

**Le pari honnête (plafond réaliste)** : pas un « substrat », mais un **« WireGuard pour agents
multi-éditeurs »** — un *sidecar* de messagerie **souverain, cross-vendor, local, chiffré** pour
essaims hétérogènes. Une **feature à culte / candidate à l'acquihire**, pas une plateforme venture.

---

## 4. TÂCHES À FAIRE — « TOUT FAIRE » (priorisé)

> Ordre = impact/effort + irréversibilité (faire les irréversibles-gratuits d'abord, tant qu'Aïssa solo).

### A. Gains gratuits / fenêtre solo (faire EN PREMIER)
- [ ] **M3 — Relicence MIT → Apache-2.0** : remplacer `LICENSE`, mettre à jour les en-têtes/`.goreleaser`
  (`license: Apache-2.0`), `README`/`PROJECT.nfo`. **Coordonner avec surface:62** (elle a posé le MIT
  partout). Ajouter **NOTICE** + un **CLA** léger (`docs/CLA.md`) avant tout PR externe. *Garde-fou :
  ne casser aucune attribution ; commit dédié, message clair.*
- [ ] **M3bis — Marque** : lancer une recherche de disponibilité « csend » (le nom est **faible** :
  imprononçable, collision SEO avec `send()`). Décider rebrand ou non (voir §5 « marque » du doc). Si
  rebrand : `.dev`/npm/brew/marque à clear AVANT (coût ~0 à v0.2.0, 301 + alias transitoire).
- [ ] **M1 — Provenance** : ajouter au workflow release **SLSA L3** (slsa-github-generator), **cosign**
  keyless (signature + attestation, Rekor), **SBOM CycloneDX**, builds reproductibles. Publier un
  **threat model agentique** (`docs/THREAT-MODEL.md`, STRIDE/MAESTRO) — honnête : « PQC implémenté,
  **non audité** » (§29/§34, zéro fausse preuve). *Bloqueur : la CI est désactivée tant que la
  facturation GitHub n'est pas régularisée → faire tourner en local / préparer le workflow prêt.*

### B. La niche (le vrai produit) — 2ᵉ vague d'agents recommandée
- [ ] Lancer une **2ᵉ exploration ciblée** (agent teams) sur **uniquement** le pari retenu
  (« WireGuard for multi-vendor agents ») : design technique (preuve que Claude+Codex+Gemini parlent
  dans un même bus local), GTM précis (qui, où, comment), preuve/démo virale, positionnement. *Réutiliser
  le script `…/workflows/scripts/csend-deep-exploration-*.js` comme base (vagues de 3 + ré-essai).*
- [ ] Construire la **démo cross-vendor** réelle (un Codex + un Gemini + un Claude qui collaborent via
  csend) = la preuve du créneau, et le meilleur asset marketing (dogfooding public).

### C. Hygiène / distribution (en partie surface:62)
- [ ] Moderniser les **2 dépréciations goreleaser** (`archives.format`/`brews` → format moderne) avant
  la prochaine release.
- [ ] **Homebrew tap** : créer `aissablk1/homebrew-tap` (PAT scope repo) + relancer `goreleaser release`
  → `brew install aissablk1/tap/csend`. *(compte d'Aïssa)*
- [ ] **Site** : déployer `site/` sur **Cloudflare Pages → csend.dev** ; mettre l'URL réelle dans
  `index.html` + `install.sh`. *(compte d'Aïssa)*
- [ ] **CI verte** : régulariser la **facturation GitHub** (débloque Actions ; les binaires sont déjà
  publiés via goreleaser local). *(compte d'Aïssa)*
- [ ] **Diffusion** (seulement après la démo cross-vendor) : awesome-lists (claude/go), Show HN, r/ClaudeAI,
  X (vidéo `brag-output/`). *Positionner « sidecar souverain cross-vendor », pas « substrat » (§34/§29).*

### D. Durcissement code (optionnel, mon territoire)
- [ ] **Transcript signé incluant l'ID** (lier expéditeur+destinataire+timestamp dans la signature) →
  anti-replay total (bloque un payload re-emballé sous un nouvel ID). Fichier `crypto.go`.
- [ ] **Audit crypto** AVANT toute communication « production / Signal-grade » (sinon survente §34).

---

## 5. GARDE-FOUS & RÈGLES ACTIVES (rappels)
- **§29** : faits vérifiés (la critique a démonté la synthèse — ne pas survendre). PQC **non audité**.
- **§34** : zéro fausse preuve sur le site/la diffusion (pas de « substrat », pas de perf inventée).
- **§7** : sessions parallèles, stage par chemin explicite, ne pas écraser surface:62.
- **§5/§38** : pas de secret côté client ; ports `serve` jamais hors loopback sans `--authz` ; PAT/clé en vault.
- **§35** : email perso jamais exposé ; authorship = GitHub `noreply`, contact public = `contact@aissabelkoussa.fr`.
- **§57** : la bonne réponse est souvent soustractive — la niche étroite **est** le bon résultat, pas un échec.

---

## 6. RÉFÉRENCES
- Repo : github.com/aissablk1/csend (public, MIT, release v0.2.0).
- Exploration complète : `docs/exploration/2026-06-29-csend-99pct-exploration.md`.
- Spec/plan : `docs/superpowers/specs/2026-06-27-csend-bus-universel-design.md`, `docs/NEXT.md`, `docs/PUBLISHING.md`, `docs/adoption/`.
- Mon commit de fix : `b716daf`. HEAD au handoff : `37924d6` (42 commits).
- Script d'exploration réutilisable : `~/.claude/projects/…/workflows/scripts/csend-deep-exploration-wf_24cd8cf1-b6f.js`.

---

## 7. Journal d'exécution — 2026-06-30 (GROUPE A livré : le primitif gardé)

Décision Aïssa « tous en autonomie » → j'ai construit le **seul** survivant des explorations
(le primitif *inject & receive* cross-vendor), dans mon territoire (zéro conflit §7), testé + commité :

- **Spike Jour-1** (cross-vendor) : pipe `gemini→csend→hook Claude` **prouvé sub-seconde** ; le vrai
  verdict Gemini est **bloqué** (Gemini CLI non authentifié → besoin `GEMINI_API_KEY` d'Aïssa). Confirme
  la critique : le transport est trivial, le « wow » autonome dépend de l'auth.
- **commit 44** — `csend journal` (P-demo) : trace du bus en **hash** (jamais le clair), money-frame.
- **commit 45** — enveloppe **provider-aware** (`provider`/`kind` sur `InboxMessage`, auto-peuplée) →
  `csend journal` affiche `gemini:gemini-rev → claude-dev` (cross-vendor **visible**).
- **commit 46** — **`csend hook` provider-aware** : lit le payload stdin, **dérive l'identité du
  session_id** (zéro-config, fini `CSEND_AGENT_ID` manuel), émet la bonne forme (Claude
  `hookSpecificOutput` / Gemini brut). Prouvé en smoke.
- **commit 47** — **démo `scripts/demo-cross-vendor.sh`** « Green Build Relay » : 3 éditeurs
  (claude/codex/gemini) sur **un** bus, relais réel claude→codex→gemini→claude + money-frame. Contenu
  scripté (driven), **honnête** §2/§29.
- **commit 48** — `csend hook --install claude|codex|gemini` (snippets par éditeur, pur affichage §5).

### 2026-07-01 (GROUPE C + M3 + M1 partiel — tout POUSSÉ sur origin/main)

- **commit 49** — **#41 crypto durci (BREAKING → v0.3.0)** : le transcript signé+AEAD lie
  désormais `SenderPub` + une **AAD `from→to`** ; un payload ré-emballé sous un autre couple
  expéditeur→destinataire est **rejeté**. API : `Seal`/`Open` prennent un `aad ...[]byte`
  variadique (appels existants intacts) ; `maybeSeal`/`openBody`/`senderAllowed` passent
  `sealAAD`. Nouveau test `TestOpenRejectsRewrappedUnderDifferentIdentities`. Suite verte.
- **commit 50** — **M1 (volet honnêteté)** `docs/THREAT-MODEL.md` : STRIDE + limites assumées
  (**crypto NON auditée**, même-UID = crypto de peu de valeur, prompt-injection hors périmètre
  du bus, métadonnées exposées). Zéro fausse promesse (§29/§34).
- **commit 51** — **M3 relicence MIT → Apache-2.0** (#37) : grant de brevet (vital vu le PQC),
  recommandé par l'exploration. `LICENSE` = texte canonique Apache-2.0 ; `NOTICE` ; `docs/CLA.md`
  (DCO léger) ; refs MAJ dans `.goreleaser.yaml`/`README.md`/`PROJECT.nfo`. **Réversible.**
- **PUSH** : les 9 commits (44→51) sont **publics** sur `origin/main` (`d069531`). surface:62
  **prévenue** par csend (breaking v0.3.0 + relicence) pour le prochain release.

### 2026-07-01 (GROUPE D — finition réelle + convergence surface:62)

- **commit `54087d0`** — **aide complète** : `hook`/`watch`/`journal`/`key` étaient absents de
  `csend help` (découvrabilité) → ajoutés. **+ `csend hook --provider claude|codex|gemini`**
  (override quand le CLI ne passe pas `hook_event_name`). Complète P0 honnêtement. Suite verte.
- **Convergence surface:62** : sur ma note csend, elle a poussé **`bbb3efe`** = « finir relicence
  Apache-2.0 côté distribution (site, PUBLISHING, playbook, Formula) ». → **repo 100 % cohérent
  Apache-2.0** (seuls `RELEASE-v0.2.0.md`/`brag-output`/`dist` gardent MIT, historiquement correct).
  **Deux sessions convergées** dans `origin/main` (`54087d0`), tests verts, sync 0/0.

### 2026-07-01 (GROUPE E — combler les vraies lacunes, zéro semblant)

- **`docs/cross-vendor-setup.md`** — guide réel : câbler Claude+Codex+Gemini sur un bus csend
  (hook par éditeur, motif `/second-opinion`, E2E optionnel), prérequis honnêtes (auth Gemini,
  install+trust Codex, store partagé). Le P0/P1 était codé mais **sans mode d'emploi** → comblé.
- **CHANGELOG `[Non publié]`** — était stale ; documente maintenant le **breaking crypto**, la
  **relicence Apache-2.0** et les nouvelles commandes (vers 0.3.0).
- **`hook --provider` vérifié** (smoke correct) : Claude→JSON, Gemini→brut, auto-identité OK.
  L'échec précédent était une erreur de mon test, pas du code (pas de fausse affirmation).
- **Test d'intégration `TestNetworkE2ESealedRoundtrip`** — vraie lacune : `net_test` couvrait le
  transport en clair, pas un message **scellé** ; ajoute le round-trip E2E sur TCP (le chiffré
  survit + l'aad §41 est préservée sur le fil). **PASS**, suite `-race` verte. Poussé.

### 2026-07-02 (billing élucidé + SBOM local)

- **Correction §29** : le « CI bloqué par la facturation » est **confirmé** (annotation explicite :
  *« job was not started because your account is locked due to a billing issue »*). Les Actions sont
  activées mais le **compte** est verrouillé → jobs non démarrés. Fix = **Aïssa** sur
  `github.com/settings/billing`. (J'avais flip-floppé en séance — la réalité tranche.)
- **commit `4ce08bd`** — **SBOM CycloneDX local** dans `.goreleaser.yaml` (part de #38 faisable sans
  Actions, comme la release v0.2.0 locale ; `goreleaser check` valide le bloc ; `syft` requis à la
  release). **SLSA-in-Actions + signature cosign** restaient gatés.
- **FACTURATION DÉBLOQUÉE** (Aïssa a mis une carte Visa valide) → **CI VERTE** (`success`, 3 OS +
  race, vérifié via `gh run watch`). Le blocage « account locked / billing » est **résolu**.
- **commit `d20a235`** — **`.github/workflows/release.yml`** (#38 COMPLET côté Actions) : sur tag
  `v*`, goreleaser (binaires + SBOM) → **attestation SLSA native** (`actions/attest-build-provenance`)
  → **signature keyless cosign** des checksums (OIDC). YAML validé ; **se prouve au tag v0.3.0**
  (surface:62 le coupe — prévenue : release via Actions, pas de goreleaser local en parallèle).
- **#38 = FAIT** : threat-model + SBOM + SLSA + cosign. Ne reste qu'à l'exercer au 1er tag v0.3.0.

### 2026-07-02 (GROUPE G — #40 REBRAND csend → Communikey)

- **Décision Aïssa** (après plusieurs vagues de vérif dispo npm/GitHub/domaine) : nom **Communikey**
  (communiqué + key ; npm libre, communikey.dev libre), CLI `communikey` + **alias `comkey`**, env
  vars `COMKEY_*`.
- **Fait + poussé (mon côté)** : repo GitHub **renommé `aissablk1/communikey`** (redirections), remote
  MAJ ; **code** (30 fichiers, module `github.com/aissablk1/communikey`, `COMKEY_*`, domaines HKDF
  `communikey/...`) — **build + tests verts** ; **mes docs** (NOTICE, THREAT-MODEL, cross-vendor-setup,
  CLA, PROJECT.nfo dont l'ASCII), **service/** (unités renommées), Makefile, .gitignore, NEXT.md ; le
  **skill** `~/.claude` → `/communikey`. Breaking crypto (nouveaux domaines) cohérent v0.3.0.
- **Coordonné à surface:62** (distribution) : sweep de README, site/, `Formula/csend.rb` (→ `communikey.rb`),
  CHANGELOG `[Non publié]`, docs/adoption+strategy ; **alias `comkey`** (symlink dans install.sh + Formula) ;
  **domaine communikey.dev** à réserver.

**Reste (par groupes) :**
- **GROUPE B** (après clôture de surface:62, §7) : **M3** relicence Apache-2.0 (#37) + **M1** provenance
  SLSA/cosign/SBOM + `THREAT-MODEL.md` (#38). surface:62 a journalisé (commit `c5fdba5`) mais est
  encore ouverte → attendre.
- **GROUPE C** : **#41** transcript signé incluant l'ID (breaking → v0.3.0, délibéré) ; **#40** marque
  (décision Aïssa) ; raffinement démo : échanger les contacts E2E pour que le money-frame affiche
  « chiffré » au lieu de « clair ».
- **Bloqué sur Aïssa** : `GEMINI_API_KEY` (pour le vrai spike wow + la démo à verdicts d'IA réels).

**Auteur** : Aïssa BELKOUSSA
