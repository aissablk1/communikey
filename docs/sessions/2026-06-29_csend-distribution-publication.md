---
session_id: bb751108-3b17-431b-ac12-a99fd564397e
date_debut: 2026-06-27T20:12:32Z
date_fin: 2026-06-30
workspace: csend (/Volumes/Professionnel/Projets/Développement/Outils/csend)
auteur: Aïssa BELKOUSSA
statut: clôturé
tags: [csend, distribution, publication, release, homebrew, goreleaser, oss]
---

# Session — csend : viabilité, distribution & publication v0.2.0

> Session « distribution » (surface parallèle à la session « cœur » surface:45, qui a livré
> hook/watch, `--authz`, anti-replay, verrou fichier, CI cross-OS, service launchd/systemd).
> Territoires **disjoints** ; convergence via `origin/main`. Journal cœur partagé :
> [`2026-06-27_csend-bus-universel.md`](2026-06-27_csend-bus-universel.md).

## QQOQCCP

- **Qui** : Aïssa BELKOUSSA (assistant en autonomie), en coordination avec la session surface:45.
- **Quoi** : évaluer la viabilité de csend, puis le rendre **publiable et installable** (open source).
- **Où** : repo `github.com/aissablk1/csend` (+ tap `aissablk1/homebrew-tap`).
- **Quand** : 2026-06-27 → 2026-06-30.
- **Comment** : analyse marché → build multi-agents → calibration sur source → publication
  (repo public, release goreleaser locale, tap Homebrew) → coordination inter-sessions.
- **Combien** : 9 commits de distribution + 1 release (7 assets) + 1 tap ; 14 tasks closes.
- **Pourquoi** : faire de csend un outil OSS trivial à installer (humains + agents) et honnête.

## Actions analysées

- **Viabilité (/startup + /brainstorming)** : recherche marché réelle — marché agents IA ~8 Md$
  (2026), « année de l'orchestration ». **Constat décisif** : Anthropic **Agent Teams** (natif,
  févr. 2026) duplique le cœur « messagerie inter-sessions » ; champ encombré (ruflo, tmux-orchestrators)
  + protocoles standards (A2A/MCP/ACP). → csend n'est PAS viable comme « orchestrateur » générique.
- **Verdict (choix d'Aïssa)** : **viable en open source, pas en business**. Moat défendable =
  souveraineté + chiffrement E2E/PQC + cross-provider + cross-OS, là où le natif ne va pas.

## Actions réalisées

- **Playbook d'adoption OSS** : `docs/adoption/playbook-adoption-oss.md` (positionnement, canaux,
  friction d'install, KPI, plan 30 j).
- **Build « tout réaliser » (4 agents parallèles, chemins disjoints)** :
  1. **Docs/confiance** : README v2 (comparatif honnête, modèle de sécurité, matrice OS, roadmap,
     badges), `SECURITY.md`, `CHANGELOG.md`, **`LICENSE` MIT** (manquait alors que revendiqué partout),
     `Formula/csend.rb`, `RELEASE-v0.2.0.md`. → commit `e3dfc23`.
  2. **Cross-provider** : `adapters.go` + `adapters_test.go`, registre `provider.go`. → commit `3bac2fd`.
  3. **Site landing** : `site/` direction « terminal souverain », anti-slop, WCAG AA, zéro fausse
     preuve. → commit `eb13d6c`.
  4. **Vidéo de lancement** : `brag-output/brag.mp4` (sans audio → publiable, §60).
- **Calibration adaptateurs sur SOURCE PRIMAIRE (§2/§29)** : Gemini sur le bundle installé
  `@google/gemini-cli` 0.40.1 (grep local), Codex sur `openai/codex` `rust-v0.142.3` (fetch
  contre-vérifié). Suppositions fausses retirées (`context left`, `gpt-\d`, `⏎ send`). Tests verts.
- **Prep publication** : `csend version` (+ injection ldflags), `_backup/`/`brag-output/`/`dist/`
  ignorés, statut Codex/Gemini à jour (README + site). Commits `4c51d4c`, `e3dfc23`.
- **Revert effet de bord §5** : `hyperframes init` (lancé pour la vidéo) avait semé 72 liens dans
  9 outils d'agents tiers + 8 dossiers dans `~/.agents`. Révertés avec manifestes de backup ;
  `~/.claude` et le store `~/.agents` préservés (choix d'Aïssa).
- **Publication** : **repo PUBLIC** créé + `main` poussé (pre-flight secrets OK). → `go install` opérationnel.
- **Release binaires v0.2.0 (goreleaser local)** : `goreleaser` installé (`go install`), tag `v0.2.0`,
  `goreleaser release` → **6 binaires** (darwin/linux/windows × amd64/arm64) + `checksums.txt` +
  **GitHub Release publiée via l'API** (contourne les Actions verrouillées par la facturation).
  Vérifié : binaire `csend 0.2.0`, `releases/latest/download/…` → HTTP 200.
- **Homebrew** : config goreleaser modernisée (`archives.formats` ; **formule** conservée vs cask
  macOS-only) → commit `63fc386` ; **tap `aissablk1/homebrew-tap` créé** + **formule binaire
  cross-plateforme** poussée (mac+Linux, intel+arm, 4 url/sha256).
- **Polish** : badge release + install au présent dans le README → commit `37924d6`.
- **Coordination inter-sessions (§7)** : convergence via `origin/main` avec surface:45 ; territoires
  disjoints ; `pull --rebase` avant chaque push pour rester fast-forward. Commits `c51bcd3`, `dfb7e66`,
  `1381242`.

## Actions à mener à l'avenir (bloquées → guidées à Aïssa)

- **Régulariser la facturation GitHub** : verrou de compte = Actions OFF → bloque la **CI** (badge
  rouge, code pourtant vert) ET **GitHub Pages**. Blocage racine unique.
- **Site en ligne** : Cloudflare Pages (§58, indépendant de la facturation) ou GitHub Pages après
  déblocage ; puis domaine `csend.dev` (DNS).
- **Diffusion** : Show HN, r/ClaudeAI, X (vidéo `brag-output/`) ; PRs awesome-lists **plus tard**
  (alpha d'un jour = risque de rejet).
- **Adaptateurs** : confirmation **live** (auth Gemini / install Codex) pour valider la calibration source.
- **Homebrew** : `homebrew_casks` à réévaluer si goreleaser retire `brews` (perte de Linux).

## Notes / Décisions / Blocages

- **Décision** : viable en **OSS**, pas en business (doublon du natif Agent Teams).
- **Décision** : **formule** Homebrew (cross-OS) plutôt que cask (macOS-only), malgré la déprécation.
- **Astuce clé** : goreleaser **en local** publie la release via l'API → contourne les Actions
  verrouillées par la facturation. C'est ce qui a permis binaires + tap malgré le verrou.
- **Blocage** : facturation GitHub = Actions OFF (CI + Pages). Hors de portée en autonomie.
- **Honnêteté (§29)** : CI rouge = facturation, PAS le code (vert en local) ; pas de fausse preuve
  d'adoption sur le site ; calibration adaptateurs = sur source, confirmation live encore à faire.
- **Garde-fous respectés** : §5 (confinement, hook `brew install` honoré), §7 (commits file-by-file,
  pas de bulk, convergence), §21 (zéro co-auteur IA), §29/§32 (vert avant commit), §35 (email noreply).

## Addendum 2026-07-01 — Relicence Apache-2.0 (post-clôture)

- Suite à la relicence **MIT → Apache-2.0** de surface:45 (`d069531`, grant de brevet), refs MIT
  restantes de **mon territoire** passées en Apache-2.0 : `site/` (meta/og/hero/footer),
  `docs/PUBLISHING.md`, `docs/adoption/playbook`, `Formula/csend.rb` (draft). **Gardé MIT** :
  `RELEASE-v0.2.0.md` + entrées de journal — v0.2.0 *fut* publié sous MIT (exact historiquement ;
  le **tag v0.2.0 reste MIT**).
- Acté : **crypto breaking → prochaine release `v0.3.0`** (transcript signé liant
  expéditeur+destinataire, incompatible v0.2.0). Au tag v0.3.0, la release + le tap Homebrew se
  régénéreront en Apache-2.0.

## Addendum 2026-07-01 — Analyse concurrentielle « tous les axes »

- Sur ordre d'Aïssa (« surpasser ClaudeKit »), teardown (skill `competitive-teardown` + 2 agents +
  vérif source primaire). **Deux recadrages honnêtes (§29/§66)** : (1) ClaudeKit et les repos
  `claudekit`/awesome-skills = **packs de capacités intra-session** (subagents *« inspect and
  report »*), **pas** des concurrents de csend → « surpasser ClaudeKit » est un faux combat.
  (2) Le vrai concurrent = **hcom** (`aannoo/hcom`) : bus inter-sessions, 10 CLIs, cross-machine,
  chiffré E2E (PSK partagé), MIT, ~363★ — le créneau n'était **pas vide**.
- **Wedge défendable** (vérifié) : hcom admet *« sender identity is not authorization »* (PSK, pas
  d'auth d'expéditeur, pas de PQC, pas de recovery). csend seul à offrir **provenance signée
  Ed25519 + chiffrement par destinataire + ML-KEM-768 + recovery Shamir/BIP-39**. Retards assumés :
  largeur providers, intégration turnkey (hooks vs injection), maturité. Livrable :
  `docs/strategy/csend-axes-de-depassement.md`.
- **Exécution « Tous » (2026-07-01)** — MON territoire : **recadrage du message** sur le vrai wedge
  (identité cryptographique signée vs « mot de passe partagé ») — bloc ajouté au README. **Pour
  surface:45** (territoire core/crypto, §7) : (1) fermer le gap de largeur via **livraison par
  hooks** multi-CLI (à la hcom) en plus de l'injection clavier ; (2) **confirmer Codex/Gemini en
  live** ; (3) éventuelle **couche kit mince** d'orchestration inter-sessions signée. Ces 3
  mouvements sont du **code** (≠ mon territoire distribution) → à toi quand tu veux.
- **Exécution « Tous » suite (2026-07-01)** — livrables réels, mon territoire : (1) **site recadré**
  sur le wedge (kicker + lede : « signé, tu sais *qui* parle ») ; (2) **plan de lancement haute
  assurance** `docs/strategy/csend-plan-lancement-haute-assurance.md` (ICP netsec/régulé/souverain,
  canaux ORB, phases gatées ; honnête : ne pas lancer un alpha à CI rouge, pas de fausse traction) ;
  (3) **battle-card publique** `docs/COMPARISON.md` (honnête vs hcom/Agent Teams, liée au README).
  SECURITY.md laissé tel quel (déjà excellent, §57 — pas de faux travail). Bloqués → guidés : code
  (surface:45), déploiement/CI/social (comptes d'Aïssa).

## Addendum 2026-07-02 — CI verte + release durcie (coordination surface:45)

- **Facturation débloquée par Aïssa → CI VERTE** (run success). Le gate « ne pas lancer un alpha à
  CI rouge » du plan de lancement tombe.
- surface:45 a ajouté **`.github/workflows/release.yml`** : sur tag `v*`, release via goreleaser +
  **SBOM + attestation SLSA native + signature cosign keyless**. → **La release passe par Actions ;
  NE PLUS lancer goreleaser en local** (double release). v0.3.0 = `git tag v0.3.0 && git push --tags`
  (surface:45 la coupe quand la crypto est prête). Décision `brews` : **garder la formule** (cross-OS).
- **Renfort du wedge** : releases signées/attestées + SBOM = provenance supply-chain que hcom n'a pas
  → ajouté à `docs/COMPARISON.md` (honnête : à partir de v0.3.0).
- **Note d'état (§7)** : le working tree a des modifs non committées (README, site — dont un revert
  du wedge site en local) qui ne sont PAS les miennes ; **mon wedge reste committé dans HEAD** (public
  OK). Je n'y touche pas ; revert working-tree à confirmer par Aïssa/surface:45.

## Addendum 2026-07-02 — Rebrand communikey (territoire distribution)

- Aïssa a acté **csend → Communikey** (CLI `communikey` + alias `comkey`). surface:45 a fait
  code/module/binaire/env `COMKEY_*`/service/skill + renommé le repo (`aissablk1/communikey`,
  redirections). **Ma part (distribution)** : sweep `csend→communikey` / `CSEND_→COMKEY_` + alias
  `comkey` dans README, site/ (index, styles, main.js, install.sh), install.sh, Formula
  (`csend.rb`→`communikey.rb` + symlink comkey), CHANGELOG [Non publié], docs/adoption, docs/strategy,
  COMPARISON, PUBLISHING. Commit `68155f7` (13 fichiers).
- **Gardé en `csend` (historique, §29)** : CHANGELOG section [0.2.0], RELEASE-v0.2.0.md, journal.
  Backup pré-sweep dans `_backup/rebrand-communikey-…` (§6).
- **Méthode** : sed avec bypass justifié (§26) + backup + vérif grep (0 résiduel) ; symlinks/CHANGELOG
  en édits ciblés. Remote local pointé sur `…/communikey.git`.
- **À signaler (site)** : le **wedge du hero** (kicker/lede « signé — tu sais *qui* parle ») a été
  **réverté** dans le working tree (flaggé « intentionnel ») ; non réimposé. Le wedge reste dans
  README + COMPARISON. À re-ajouter au site si voulu.
- **Domaine** : `communikey.dev` à réserver (csend.dev à migrer) — compte d'Aïssa. Vérifié
  **disponible** (Vercel : 9,99 $/an ; Aïssa réserve à 12,20 $). `.com`/`.app` pris.
- **Finition (feu vert Aïssa)** : wedge « signé » **restauré** sur le hero du site (direction
  *terminal souverain* §10 ; `.reveal` + `prefers-reduced-motion` déjà dans le CSS ; typo §20) ; 2
  docs `docs/strategy/csend-*.md` **renommés** `communikey-*.md` (R100). Commit `1e65f50`.
  **SECURITY.md** aussi swept (`1b9031a`, mon fichier via e3dfc23). → **Territoire distribution
  100 % communikey.** Restent à surface:45 avant tag v0.3.0 : `docs/superpowers/{plans,specs}` +
  `testdata/claude_busy_real.txt` (core/code). Ship différé (fonds).
