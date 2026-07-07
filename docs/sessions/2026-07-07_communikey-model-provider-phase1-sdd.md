---
session_id: N/A (hook §23 non déclenché — le workspace de CETTE session Claude
  Code est $HOME, explicitement exclu par le hook `session-journal-qqoqccp.sh` ;
  journal tenu manuellement, à la demande explicite d'Aïssa)
date_debut: 2026-07-05 ~20:26 (brainstorming initial du sujet communikey/LMP)
date_fin: en cours
workspace: /Volumes/Professionnel/Projets/Développement/Outils/communikey/.worktrees/model-provider-phase1 (worktree isolé, branche feat/model-provider-phase1 ; cwd réel de la session Claude Code = /Users/aissabelkoussa)
auteur: Aïssa BELKOUSSA
statut: en_cours
tags: [communikey, feature, model-provider, sdd, tdd, go]
---

# Session — communikey : client multi-provider de modèles (Phase 1), brainstorm → spec → plan → SDD

## QQOQCCP

- **Qui** : Aïssa BELKOUSSA. Session solo dans un worktree isolé (branche
  `feat/model-provider-phase1`, créé depuis le HEAD local — `origin/main` avait
  21 commits de retard, dont la migration Argon2id sur laquelle ce travail
  s'appuie). Une autre session travaillait la veille sur `crypto.go`/ML-DSA
  (déjà committé sur `main` avant le début de ce worktree, aucun conflit).
- **Quoi** : donner à communikey une commande `model` capable d'appeler
  directement un backend d'inférence (Ollama, LocalAI, HuggingFace…) déclaré
  dans `~/.claude/communikey/models.json`, sans toucher à l'orchestration de
  sessions CLI existante. Déclenché par une recherche GitHub sur le
  « Language Model Protocol » d'Apple et son rapport (aucun) avec communikey,
  puis une proposition maximaliste d'Aïssa (tous providers + longue liste
  crypto) — recadrée via brainstorming (plusieurs éléments crypto corrigés
  factuellement : RSA-1024/SHA-1/MD5 cassés, Shor/GNFS sont des algorithmes
  d'ATTAQUE pas de défense, Coppersmith-Winograd/Busy Beaver/classification
  des groupes simples hors-sujet, FairPlay/UPX+AES mauvais modèle de menace).
- **Où** : `docs/superpowers/specs/2026-07-06-communikey-model-provider-design.md`
  (spec, commits `60372d9` + correctif `d79ae5a`), `docs/superpowers/plans/
  2026-07-06-communikey-model-provider-phase1-plan.md` (plan, 6 tâches),
  exécuté via `superpowers:subagent-driven-development` dans ce worktree.
- **Quand** : spec/plan écrits et committés sur `main` le 2026-07-06 ; exécution
  SDD démarrée le 2026-07-07.
- **Comment** : brainstorm → spec (validé, committé) → plan TDD détaillé (6
  tâches, code complet, auto-relu) → exécution subagent-driven (1 implémenteur
  frais + 1 reviewer par tâche, boucle de fix si trouvaille Important/Critical,
  ledger de progression `.superpowers/sdd/progress.md`). Modèles : implémenteurs
  en Haiku (code déjà entièrement écrit dans le plan = transcription+tests),
  reviewers en Sonnet.
- **Combien** : 3 tâches terminées et approuvées à ce stade (sur 6) + 2 fix +
  2 re-revues ; 9 commits sur la branche à ce stade.
- **Pourquoi** : répondre à un vrai besoin (consommer des modèles multi-provider
  depuis communikey) tout en évitant la dérive constatée dans la demande
  initiale (liste crypto en partie erronée, portée « tout le marché d'un
  coup ») — via le pipeline brainstorm→ship (§30) plutôt qu'une exécution
  directe sur la demande brute.

## Actions analysées (réflexion / diagnostic)

- [2026-07-06] Recherche GitHub vérifiée (`gh api`) : protocole `LanguageModel`/
  `LanguageModelExecutor` d'Apple (WWDC 2026), implémentations réelles
  (`apple/coreai-models`, `ml-explore/mlx-swift-lm`, `anthropics/
  ClaudeForFoundationModels`, `huggingface/AnyLanguageModel`) — aucun rapport
  technique avec communikey (couches différentes : appel de modèle vs
  orchestration de sessions CLI par lecture d'écran).
- [2026-07-06] Correction factuelle de la liste crypto proposée par Aïssa avant
  tout design (voir Quoi ci-dessus) — décomposition en 2 sous-projets
  indépendants (consommation de modèles / crypto), seul le premier retenu.
- [2026-07-06] 3 questions de clarification (QCM) : extension de communikey
  (pas nouveau produit) ; les 3 usages voulus à terme (workers=nœuds du bus,
  hooks enrichis, commande utilitaire) ; local+cloud dès le départ avec
  garde-fous opt-in.
- [2026-07-06] Lecture directe du code (`provider.go`, `providerconfig.go`,
  `keyring.go`, `crypto.go`, `main.go`, `jsonout.go`) pour ancrer le design
  dans la réalité plutôt que dans la doc — a révélé que le spec initial
  mentionnait PBKDF2/`keyring.go` alors que le code avait déjà migré vers
  Argon2id (`SealVault`/`OpenVault` génériques dans `crypto.go`, commit
  `eb73942` concurrent) — corrigé avant d'écrire le plan.

## Actions réalisées (exécution / livrables)

- [2026-07-06] Spec écrit et committé : `docs/superpowers/specs/
  2026-07-06-communikey-model-provider-design.md` (commit `60372d9`), corrigé
  (`d79ae5a`).
- [2026-07-06] Plan TDD écrit (6 tâches, code complet) et committé (avec un
  correctif ultérieur) : `docs/superpowers/plans/
  2026-07-06-communikey-model-provider-phase1-plan.md`.
- [2026-07-07] Worktree isolé créé manuellement (`git worktree add`, PAS l'outil
  natif `EnterWorktree` — son défaut `baseRef: fresh` serait parti
  d'`origin/main`, 21 commits en retard) : `.worktrees/model-provider-phase1`
  sur branche `feat/model-provider-phase1`. `.gitignore` mis à jour (commit
  `a76fbfb`). Baseline vérifiée : build + tests verts avant toute modification.
- [2026-07-07] **Task 1** — parsing déclaratif de `models.json`.
  - Fichiers : `modelconfig.go` (+45), `modelconfig_test.go` (+57).
  - Commit : `b9fa2b4 feat(model): parsing declaratif de models.json`.
  - Revue : ✅ Spec compliant, Approved. 3 trouvailles Minor (cosmétiques,
    héritées du snippet du plan lui-même) consignées au ledger, aucun fix
    nécessaire.
  - Incident opérationnel : le subagent implémenteur (Haiku) a planté sur son
    message de statut final (« Prompt is too long », erreur API) — APRÈS avoir
    committé correctement. Vérifié indépendamment (diff conforme au plan au
    caractère près, `go build`/`go test` verts) avant de passer en revue.
- [2026-07-07] **Task 2** — interface `ModelProvider` + adaptateur compatible
  OpenAI.
  - Fichiers : `modelprovider.go` (+31), `modelclient_openai.go` (+113),
    `modelclient_openai_test.go` (+70).
  - Commit : `a5203bd feat(model): adaptateur ModelProvider compatible OpenAI`.
  - Même incident opérationnel (crash Haiku sur le message final, post-commit) —
    revérifié indépendamment (diff conforme au plan, build/vet/test verts).
  - Revue : ✅ Spec compliant, mais 1 trouvaille **Important, plan-mandated** :
    la branche « erreur JSON dans un body HTTP 200 » n'était testée par aucun
    des 3 tests prescrits par le plan lui-même. **Décision Aïssa (QCM)** :
    corriger maintenant plutôt que laisser en dette.
  - Fix : `2ffefb2 test(model): couvre le champ error JSON en reponse HTTP 200`
    — 1 test ajouté, aucune autre modification. Vérifié indépendamment
    (diff minimal, build/test verts).
  - Re-revue : ✅ Approved, 0 trouvaille.
- [2026-07-07] **Task 3** — secrets de provider via le vault existant.
  - Fichiers : `modelsecret.go` (+121), `modelsecret_test.go` (+83).
  - Commit : `15481da feat(model): secrets de provider via le vault existant`.
  - 3e incident opérationnel identique (crash Haiku sur le message final) —
    cette fois le rapport avait aussi été écrit au mauvais chemin (racine du
    worktree), déplacé par le contrôleur. Commit et code vérifiés
    indépendamment (diff conforme au plan, build/vet/test verts) avant revue.
  - Revue : ✅ Spec compliant (zéro nouvelle primitive crypto confirmée —
    tout délègue à `SealVault`/`OpenVault`/`resolveVaultPass` existants), mais
    1 trouvaille **Important, plan-mandated** : le comportement « fusionne
    sans écraser » de `saveModelSecret` était correct mais totalement non
    testé. **Décision Aïssa (QCM)** : corriger maintenant (cohérent avec la
    Task 2).
  - Fix : `1752a1d test(model): couvre la fusion des secrets sans ecrasement`
    — 1 test ajouté, aucune autre modification.
  - Re-revue : ✅ Approved, 0 trouvaille.

- [2026-07-07] **Task 4** — `buildModelRegistry()` (registre déclaratif).
  - Fichiers : `modelprovider.go` (+50, append pur), `modelprovider_test.go`
    (+82, nouveau).
  - Commit : `ac801ba feat(model): construction du registre de providers de
    modele`.
  - **Basculé sur Sonnet** (plutôt que Haiku, utilisé pour Tasks 1-3) vu le
    pattern de crash récurrent — aucun crash cette fois, complétion propre.
  - Revue : ✅ Spec compliant, Approved, 0 fix nécessaire. 3 trouvailles Minor
    (plan-mandated, prospectives) consignées au ledger.

## Actions à mener à l'avenir (TODO / follow-up)

- [ ] Task 5 — sous-commandes CLI `model list|test|call|secret set` +
  câblage `main.go`.
- [ ] Task 6 — vérification finale (`go vet`/`build`/`test`) + CHANGELOG/README.
- [ ] Revue finale de branche (modèle le plus capable) + `superpowers:
  finishing-a-development-branch`.
- [ ] Phases 2 (hooks enrichis) et 3 (workers-comme-nœuds-du-bus) — hors
  scope de ce plan, chacune sa propre spec, une fois ce socle utilisé en
  conditions réelles.

## Notes / Décisions / Blocages

- **Aucun blocage réel à ce stade.** Les trois « échecs » de subagents notés
  ci-dessus (Tasks 1, 2, 3 — pattern désormais récurrent, 100 % des
  implémenteurs Haiku dispatchés jusqu'ici) sont des plantages du **message de
  statut final** (erreur API « Prompt is too long »), jamais du travail
  lui-même — systématiquement revérifiés par le contrôleur (diff + build +
  test) avant de poursuivre, jamais pris pour argent comptant. Les fix
  subagents (Sonnet-scope, tâches plus petites) n'ont pas reproduit ce crash.
- **Décision Aïssa répétée deux fois (QCM)** : toute trouvaille « Important,
  plan-mandated » (trou de test sur un chemin d'erreur/comportement du plan
  lui-même) est corrigée immédiatement plutôt que laissée en dette — cohérent
  sur Tasks 2 et 3, à reproduire pour les tâches restantes si le cas se
  représente.
- **Décision actée** : liste crypto de la demande initiale d'Aïssa
  explicitement écartée du design (RSA-1024/SHA-1/MD5 cassés ; Shor/GNFS sont
  des attaques, pas des défenses ; le reste hors-sujet) — aucun de ces
  éléments n'apparaît dans le code livré.
- **Dette documentée (Minor, non bloquante)** : `modelconfig.go` non
  `gofmt`-clean (hérité du plan, CI ne vérifie pas gofmt) ; contenu vide sur
  `choices[0].message.content` non distingué d'un échec (Minor, plan-mandated,
  cf. revue Task 2) ; absence de test sur l'en-tête Authorization absent.
  Toutes consignées pour la revue finale de branche.

---

**Auteur** : Aïssa BELKOUSSA
