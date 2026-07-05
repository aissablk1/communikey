# csend — Plan de la niche « WireGuard for multi-vendor agents »

- Date : 2026-06-30
- Axes : 8/8 (faisabilité, ICP, use-case, GTM, moat, concurrence, démo, kill-criteria)
- Méthode : agent teams (vagues de 3 + ré-essai) → plan → critique GO/NO-GO
- Auteur : Aïssa BELKOUSSA

---

## Plan

# Feuille de route csend — « WireGuard pour agents multi-éditeurs »

Aïssa, voici le plan, synthétisé à partir des 8 analyses et débarrassé du gras. Le fil conducteur : le marché est **petit et déjà contesté** (AWS CAO, hcom, agent-link-mcp, ruflo, Agent Teams natif, A2A) — donc on ne « lance » pas un produit, on **valide une hypothèse étroite en 6 semaines**, avec un seul angle défendable et une démo qui le rend visible en 30 secondes.

**Verdict de cadrage (à accepter avant de coder)**
- La **réception cross-vendor est déjà résolue** : les 3 CLI ont convergé sur le même modèle de hooks (`UserPromptSubmit` / `Stop`, JSON stdin/stdout, `hookSpecificOutput.additionalContext`). C'est du **câblage**, pas de la R&D.
- La **crypto PQC sur localhost est un passif**, pas un moat — à reléguer au mode remote.
- Le **seul différenciateur qui mappe sur un acheteur** : binaire Go unique, zéro dépendance, zéro télémétrie, auditable, air-gappable, **BYO-session (aucun proxy de token)**.
- Le **seul « wow » qu'un concurrent ne montre pas** : la review adversariale **cross-vendor en direct** (Agent Teams est mono-vendor → angles morts partagés, ton §41).

---

## 1. Positionnement (la promesse en une phrase)

> **csend : le seul binaire Go autonome — zéro dépendance, local, chiffré — qui fait collaborer Claude Code, Codex et Gemini en direct dans la même session ; la « seconde opinion » multi-éditeur qu'Agent Teams, mono-vendor, ne peut pas donner.**

Ce qu'on **abandonne explicitement** (§57 — soustraction) : « substrat de l'Internet des agents », « bus multi-vendor » en tête de gondole (commodisé par hcom/CAO), et la crypto post-quantique comme argument de lancement.

---

## 2. ICP + use-case tueur

**Deux cercles concentriques, à ne pas confondre :**

| | Cible | Rôle dans le plan |
|---|---|---|
| **Cercle 1 — atteignable la semaine prochaine** | Le power user **multi-vendor en terminal brut** (Claude Code + Codex/Gemini CLI lancés en parallèle), pas dans Zed/JetBrains | Validation d'usage + distribution + démo |
| **Cercle 2 — solvable, à valider via 1 design partner** | Ingénieur **plateforme/SecOps** en org **régulée à egress restreint** (fintech mid-tier, santé, sous-traitant gov **non-SCIF**) que sa sécurité a **disqualifié** des orchestrateurs npm/pip lourds | Test de WTP réelle (support/audit/signature de questionnaire) |

**On renonce frontalement** au pur SCIF/défense (procurement 12-24 mois, inatteignable solo) et aux power users « gratuits » comme cible de revenu.

**Use-case tueur retenu : `/second-opinion`** — la review adversariale cross-vendor injectée en direct.
Claude écrit → **Codex ou Gemini relit « à froid » avec instruction de RÉFUTER** (verdict ambigu = FAIL, ton §41) → le verdict **retombe dans la session Claude vivante** via le hook, sans copier-coller, sans PR, en local et chiffré. C'est la seule chose que ni les scripts bash, ni ACP (l'humain relaie), ni Agent Teams (Claude↔Claude) ne font.

---

## 3. Plan technique (dans l'ordre, ce qu'il faut coder dans csend)

L'essentiel existe déjà (bus, inbox, crypto, registre, hook Claude). Il manque **une fine couche d'adaptation par CLI sur le chemin hook** — pas un sous-système.

**P0 — `csend hook` provider-aware (pierre angulaire) · effort M**
Réécrire `cmdHook` (hook.go) en 5 temps :
1. Lire le JSON sur stdin (présent sur les 3 CLI ; ne pas l'**exiger** → rétro-compat).
2. Détecter le provider via `hook_event_name` (`UserPromptSubmit`→claude/codex, `BeforeAgent`→gemini) **ou** flag `--provider {claude|codex|gemini}` posé par l'installeur.
3. Dériver une **identité stable** depuis `payload.session_id` + `payload.cwd` (fallback `CSEND_AGENT_ID`). C'est LE gap « ça marche tout seul » : aujourd'hui il faut poser `CSEND_AGENT_ID` à la main.
4. Drainer l'inbox (déjà fait).
5. Émettre la **bonne forme stdout par provider** : Claude/Codex → `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":"…"}}` ; Gemini → l'objet d'injection de contexte `BeforeAgent`.

**P1 — Installateurs par CLI · effort M**
`csend hook --install {codex|gemini}` : merge idempotent dans `~/.codex/hooks.json` (ou `[hooks]` de `config.toml`, attention TOML) et `~/.gemini/settings.json`. **Documenter explicitement l'étape de TRUST Codex** (`/hooks` ou `--dangerously-bypass-hook-trust`) — sinon le message n'arrive jamais. Valider le JSON/TOML après écriture (jamais de config cassée, §32). Sans P1, P0 reste théorique.

**P2 — Enveloppe neutre · effort S**
Ajouter à `InboxMessage` deux champs : `From` adressable en `provider:agent` (ex. `gemini:SACEM`) + `kind` (`msg|nudge|result`). Le body scellé E2E ne change pas. Peupler `provider` **automatiquement** depuis le payload (plus de flag manuel) → `csend agents` affiche le vendor de chaque session. C'est la lingua franca qui justifie « bus neutre ».

**P-demo — `csend journal` (money-frame) · effort XS**
Mini-commande ~15-20 lignes : imprime `de → à : sha256:… (chiffré)` sans jamais le clair. Justifié par la démo, rend le chiffrement indéniable à l'image (le `journal.jsonl` interne existe déjà mais est brut).

**P3 — Déclencheur async « livrer en fin de tour » · effort L · APRÈS validation**
Côté émetteur : Claude `Stop`, Codex `Stop`+`notify`, Gemini `AfterAgent` → pousser/flusher via csend. Transforme la boîte aux lettres en coordination. Garde-fou anti-boucle (A relance B relance A) : réutiliser le `seen`/anti-replay déjà présent. Le plus délicat (3 sémantiques d'arrêt) → ne pas le faire au lancement.

**P5 — Calibration live des détecteurs Codex/Gemini · effort S · DÉPRIORISÉ**
`adapters.go` est réglé sur le code source, pas sur des écrans live. Avec les hooks universels, le chemin frappe-clavier ne sert **plus à la réception** : c'est du polish du fallback. À traiter après tout le reste.

> Prérequis machine : `npm i -g @openai/codex` (absent aujourd'hui ; claude ✅, gemini ✅).

---

## 4. La démo à construire — « Green Build Relay »

**Le seul artefact qui rend croyable le pari en 30 s.** Tâche **motivée** (chaque éditeur dans son rôle fort), zéro mock (§2), 100 % réelle, **drivée** (vrais nudges clavier — on ne simule PAS l'autonomie tant que la cadence des hooks n'est pas calibrée, §29).

**Setup (avant tournage) :** 3 panes tmux, `CSEND_AGENT_ID` stable par pane, hooks réels câblés (P0/P1). Contacts E2E échangés **avant** (sinon csend retombe en clair et la preuve s'inverse) :
```
# dans chaque pane
csend id --create
csend id --export                 # échanger les 3 jetons
csend contact add <pair> <jeton>  # 3 paires
```

**Scénario (fonction `is_prime` en TDD) :**
1. **Claude** (orchestrateur) écrit le test qui échoue + l'implémentation → `csend inbox codex "test prêt — lance pytest, dis-moi si c'est vert"`
2. **Codex** (exécuteur) : son hook fait surgir le message, lance les tests → vert → `csend inbox gemini "build vert sur is_prime — relis le diff, cas limites (0,1,négatifs)"`
3. **Gemini** (relecteur) : repère que `is_prime(1)` doit être `False` → `csend inbox claude "manque n<=1 → False"`
4. **Claude** patche. Boucle fermée.

**Money-frame de clôture (les 5 dernières secondes font 80 % de la conviction) :**
```
csend agents     # → trois providers DIFFÉRENTS sur UN bus local
csend journal    # → de → à : sha256:… (chiffré), jamais le clair
```
**Le « wow » :** on **voit** le message naître dans un pane et réapparaître dans le suivant, puis revenir — et le corps **chiffré** sur le fil. Caption : « Trois éditeurs rivaux, un bus local. Aucun cloud n'a vu un seul prompt. »

**Garde-fous d'honnêteté à l'écran (§2/§29) :** « réception via hook (voie coopérative) » ; « injection clavier live = Claude-only aujourd'hui » ; « store éphémère, vrais CLI, alpha v0.2.0 ».

**Pipeline reproductible :** dossier `demo/` + `make demo` (store `mktemp` éphémère, layout tmux, 3 vrais CLI). Enregistrement `asciinema rec` → `agg` (GIF README) + un `.tape` VHS pour le MP4 social. Prérequis : `brew install vhs asciinema agg`.

---

## 5. GTM — les 3 premiers coups de distribution

**Prérequis bloquants (à faire AVANT de tirer) :**
- **(0a) Démo + GIF en haut du README.** Sans preuve visuelle, conversion ≈ 0.
- **(0b) Décision rebrand.** « csend » est non-cherchable (collision `send`). Le Show HN et la PR awesome-list sont des **cartouches uniques** : ne pas les griller sous un nom ingouvernable. Trancher un nom qui dit la chose + garder le tagline « WireGuard for multi-editor agents ». (Tâche #40.)

**Coup 1 — PR sur `bradAGI/awesome-cli-coding-agents` · friction quasi nulle, intention maximale**
Tu te ranges à côté de hcom/AgentPipe (le public qui cherche déjà un bus d'agents). One-liner qui **différencie dès la ligne** (ne pas dire « multi-vendor bus », hcom couvre 6 éditeurs) :
> « the only **end-to-end-encrypted, single-binary, air-gappable** agent bus — cross-vendor (Claude / Codex / Gemini), zero dependencies. »
Doubler sur `hesreallyhim/awesome-claude-code` (section orchestrators).

**Coup 2 — Show HN chirurgical**
Titre mené par le différenciateur, pas générique :
> « Show HN: A single-binary, E2E-encrypted bus to make Claude Code, Codex and Gemini collaborate locally »
Mardi-jeudi ~8-10 h ET. **Premier commentaire = toi, désarmant d'honnêteté** : « hcom/AgentPipe existent et couvrent plus d'éditeurs ; CAO fait le cross-vendor sur MCP ; ma seule différence = binaire zéro-dép + BYO-session sans proxy de token, pensé pour les setups que la SecOps approuve. Et oui, la crypto est overkill en localhost — voici où elle sert. » Zéro faux upvote. Un score de 5 = un signal, pas un échec.

**Coup 3 — r/LocalLLaMA + r/selfhosted + Discord ruflo · recrutement de design partners**
Objectif des 30 jours : **5-10 utilisateurs réels** (1-2 en contexte régulé), pas des étoiles. Arguments qui portent ici : Go pur stdlib, 0 dépendance, MIT, aucun phone-home, auditable. Chasser les power users multi-agent **déjà frustrés** par le lock-in single-harness. Onboarding 1:1 de 15 min, itérer la semaine d'après.

---

## 6. Moat — défendre la niche

Sur du MIT, **aucun moat de code** (tout est forkable en un week-end). La défense = un **empilement de petits moats non-code**, dont les 3 premiers sont bon marché et expédiables :

1. **Marque « csend » + certification « csend-compatible » · M.** Le code forke, le nom non (un fork DOIT se renommer). Déposer la marque (INPI/EUIPO) ; écrire `COMPATIBILITY.md` (wire format + contrat hook + contrat provider) + un harnais `csend verify` qu'un tiers doit passer. Seul moat légalement réel.
2. **Treadmill d'intégration cross-vendor public · M.** CI matriciel testant csend contre les N dernières versions des 3 CLI + **status page publique** (« Claude Code 2.x ✓, Codex 0.y ✗ corrigé en 6 h »). Le moat n'est pas un artefact, c'est la **vitesse de réparation rendue visible** — exactement ce qu'un éditeur ne fera **jamais** pour le CLI d'un concurrent, et qu'un fork doit refaire à vie. (Branche ton §41 : le CI matriciel EST la boucle.)
3. **`csend id` web-of-trust local · M.** Web-of-trust façon SSH known_hosts / age (TOFU + signature), **sans control plane hébergé** (compatible souverain). Le switching cost = ré-appairer tous tes agents sur un fork perd l'historique de confiance accumulé. Seul effet de réseau compatible avec le positionnement.

**Hedge existentiel — « BYO-session, zéro proxy de token » · effort doc.** Les Commercial Terms Anthropic (fév. 2026) interdisent les tokens d'abonnement hors produits Anthropic (ruflo/OpenClaw bloqués le 4 avril 2026). csend **n'intercepte ni ne proxifie** les tokens — il injecte un prompt dans la session que l'utilisateur a lui-même lancée. **Documenter ce design noir sur blanc, ne jamais ajouter de proxy d'API.**

**Repositionnement, pas combat frontal :** csend = **« le dernier mile local-first et souverain de l'Internet des agents A2A »**, pas un protocole concurrent. **Geler** l'audit tiers et le pont A2A complet : ils ne se déclenchent QUE quand un design partner régulé est engagé (§57 — pas de forteresse imprenable sans garnison).

---

## 7. Kill-criteria (à figer et **dater AUJOURD'HUI**, avant tout résultat)

Anti goalpost-moving : signés maintenant, revus par un œil externe (pas d'auto-évaluation, §40/§41).

| Gate | Seuil | Échéance | Verdict |
|---|---|---|---|
| **G1 — Pain** | 30 DM ciblés → **< 5** « je le runnerais / star / paierais » | J+28 | nice-to-have → **kill** |
| **G2 — Launch pull** | Show HN + r/LocalLLaMA → **< 75 ★** ET **< 3** « I want this » non sollicités | J+14 | DOA |
| **G3 — Habitude** | **< 10** envoyeurs hebdo actifs | S+6 | pas d'usage |
| **G4 — Concurrence** | Agent Teams passe cross-vendor **OU** A2A-local couvre le messaging | continu | wedge mort → **kill immédiat** |
| **G5 — Clone-out** | wmux/agent-link-mcp passent 5k★ **et** cochent « encrypted/sovereign » | continu | clone inférieur |
| **G6 — Moat-test** | **0/15** interviews citent « chiffrement/souveraineté » comme raison de choisir | J+28 | le moat est un passif → recentrer sur use-case |

**Pré-engagement de pivot (écrit, pour ne pas re-débattre) :**
- **Pivot A (par défaut)** si G4/G6 tombent : réécrire csend en **adaptateur MCP-natif + profil A2A-local offline**, garder chiffrement/audit pour le seul segment régulé. Tu rides le standard au lieu de le combattre.
- **Pivot B** si A2A descend au niveau CLI : réduire au **seul primitif neuf** — « inject & receive into any running agent CLI » (lib Go autonome), jeter bus + crypto.

**Règle dure :** si **< 75 ★ ET < 5 « oui »** à mi-août, Pivot A est **exécuté**, pas rediscuté (anti yak-shaving §27).

---

## Séquence semaine 1 (exploitable lundi)

- **Lun-Mar** — P0 (`hook.go` : parse stdin, identité dérivée du `session_id`, sortie JSON par provider) + `npm i -g @openai/codex`.
- **Mer** — P1 (installateurs Codex/Gemini avec étape de trust) + P2 (provider/kind auto) + `csend journal` (XS).
- **Jeu** — Démo « Green Build Relay » : contacts E2E, `make demo`, asciinema → GIF + MP4.
- **Ven** — README (GIF en hero, tableau comparatif honnête csend vs CAO/hcom/ruflo, doc BYO-session) + **figer le doc kill-gates** (1 h).
- **En parallèle** — trancher le rebrand **avant** de tirer Coups 1-2 ; geler le code après jeudi pour passer en mode distribution/discovery.

Le test de survie : si un spectateur ne voit pas en **< 90 s** ce que csend fait que 50 lignes de bash ne font pas (la session Claude qui se corrige en direct sur le verdict d'un **autre** éditeur), csend lit comme « scripts + cérémonie » — et le marché minuscule se referme. Tout le travail de la semaine sert à rendre cet écart indéniable.

---

## Critique adversariale (GO / NO-GO)

Aïssa, je n'ai pas lancé de vérification web : les « faits » concurrentiels de ton plan (Agent Teams mono‑vendor, ToS Anthropic de février 2026, comptes d'étoiles, dates de blocage ruflo) sont **tes prémisses datées de 2026** que je ne peux pas re‑sourcer en primaire dans cette session. Je les prends donc comme hypothèses — et une partie du verdict porte précisément sur le fait que **plusieurs d'entre elles sont load‑bearing et non prouvées**. Voici la critique, sans gants.

---

# Critique adversariale — csend

## 0. Le tell qui contamine tout le document

Ton plan **se réfute lui‑même en page 1** et continue quand même. Tu écris noir sur blanc :

- « La réception cross‑vendor est **déjà résolue** […] c'est du **câblage, pas de la R&D**. »
- « La crypto PQC sur localhost est un **passif**, pas un moat. »
- ICP Cercle 1 = « validation, **pas revenu** » ; ICP Cercle 2 = « **à valider** ».

Traduction VC : *le différenciateur technique est du plumbing, l'argument de sécurité est admis comme inutile, et il n'existe aucun payeur prouvé nulle part dans le document.* Un fondateur honnête qui écrit ça a déjà rédigé le constat de décès — puis il propose 6 semaines de build par‑dessus. C'est le biais d'addition (§57) à l'état pur : énorme construction (P0→P5, marque, treadmill CI, web‑of‑trust) pour un besoin validé **nul**.

---

## 1. La niche est‑elle RÉELLE et atteignable, ou un mirage ?

**Réelle comme hobby. Mirage comme business.** ❌

Décompose le Cercle 1 « atteignable la semaine prochaine ». C'est l'intersection de filtres qui se multiplient, pas qui s'additionnent :

1. paie **trois** CLIs frontier rivaux,
2. les lance **en parallèle sur la même tâche**,
3. en **terminal brut / tmux**, pas dans Zed/Cursor/JetBrains,
4. ressent assez de douleur sur le copier‑coller inter‑panes,
5. pour installer un binaire Go **+ câbler 3 hooks** (dont l'étape TRUST Codex) **+ échanger des contacts E2E**.

Le produit de ces probabilités, c'est **quelques milliers de bidouilleurs mondiaux, à WTP ≈ 0**. Ce n'est pas un marché, c'est un sous‑reddit.

Le Cercle 2 (SecOps régulé) est le seul qui touche de l'argent — et c'est **une assertion, pas un segment**. Surtout : son motion d'achat n'est **pas** « un binaire MIT ». C'est SOC2, SLA de support, responsabilité contractuelle, une gorge à serrer. Une fintech mid‑tier ou un sous‑traitant gov ne « valide » pas un dev solo dont le seul livrable est un binaire et un README. Ton argument « la SecOps a disqualifié les orchestrateurs npm » se retourne : la SecOps disqualifie **aussi** « one‑man MIT project, zéro entité, zéro audit ». Tu remplaces un blocage par un autre.

Verdict Q1 : la niche existe, mais **ton propre tableau ICP prouve qu'elle n'a pas de payeur**. « Validation, pas revenu » + « à valider » = zéro revenu identifié. Atteignable ≠ monétisable.

---

## 2. Le use‑case tueur fait‑il VRAIMENT installer ?

`/second-opinion` est **la meilleure idée du document** — et reste **un vitamin, pas un painkiller**. ⚠️

Ce qui plaide pour : la review adversariale cross‑vendor mappe sur un vrai désir (seconde opinion, angles morts, ton §41 writer≠checker). C'est le seul endroit où tu touches une corde réelle.

Ce qui tue l'installation :

- **La valeur est disponible 20× moins cher.** `claude … | codex exec "réfute ça"` en **une ligne de bash**, ou je lis simplement le pane d'à côté. Tu vends « ça revient dans la session vivante sans copier‑coller » — mais l'utilisateur **regarde déjà trois panes**. Le coût cognitif que tu supprimes se compte en **secondes**, pas en friction. Tu optimises un micro‑geste pour un public qui adore les micro‑gestes manuels.
- **Le coût d'entrée est disproportionné** : binaire + 3 hooks + étape TRUST Codex (que tu signales toi‑même comme bloquante) + échange E2E + IDs stables. **20‑30 min de setup pour un “nice to have”.** Le ratio douleur‑d'installation / douleur‑résolue est inversé.
- **Le présupposé non prouvé** : que les gens *croient* que la diversité de vendor attrape plus de bugs au point de payer le setup. Plausible théoriquement, **zéro preuve** que ça déclenche un install. La plupart se contenteront du self‑review d'un seul outil.

Verdict Q2 : **théorique**. Ça génère « oh cool » sur HN et ~0 rétention. Ton propre G3 (< 10 envoyeurs hebdo = kill) est le gate le plus susceptible de se déclencher — parce que tu construis une cérémonie autour d'un geste que l'utilisateur fait déjà gratuitement à l'œil.

---

## 3. ruflo / claude‑flow ne tuent‑ils pas déjà ce pari ?

**Ils ne tuent pas l'EXPÉRIENCE `/second-opinion` ; ils écrasent le MARCHÉ.** Et ton plan **sous‑pondère gravement** cette menace : tu ne cites ruflo que comme *canal de distribution Discord* et comme *gate clone‑out G5*. C'est l'angle mort du document. ❌

Faits (que je connais en primaire via mon propre suivi de ruflo) : `ruflo` = rebrand de **claude‑flow**, **MIT, ~61 k★, très actif**, meta‑harness qui **orchestre déjà Claude Code ET Codex** (98 agents, ~314 outils MCP, swarm, hooks, daemon, mémoire). Le pattern « Claude architecte + Codex builder » — qui est *exactement* ton §40 — est **son cœur de pitch**. Donc :

- Le job « faire collaborer plusieurs agents, y compris cross‑vendor » est **déjà occupé par un incumbent massivement distribué, gratuit, MIT**.
- Le résidu différenciant de csend vs ruflo se réduit à : **binaire unique, zéro dép, E2E, air‑gappable, pas de daemon**. C'est une **pureté infra réelle mais étroite** qui ne parle **à personne** dans le public power‑user (ils lancent `npx` sans sourciller). Elle ne compte que pour le buyer régulé… **qui n'achète pas à un solo de toute façon** (cf. Q1). **Squeeze play** : tu te retrouves à courtiser le sliver qui veut de la coordination MAIS refuse npm/daemons MAIS achète à un inconnu. Cet ensemble est ≈ vide.
- **Risque de cloner inversé** : tu surveilles les petits clones (wmux, agent‑link‑mcp) en G5. Le vrai cloneur, c'est **ruvnet** : ajouter « relais in‑session + chiffrement façon csend » à claude‑flow est un week‑end pour lui — il a **déjà** les hooks et l'archi. Sur du MIT, ton seul wedge est imitable par l'acteur à 61 k★ avant que tu aies 75★.

Verdict Q3 : ruflo ne rend pas ton `/second-opinion` impossible (lui est lourd/orchestré, toi tu es live‑in‑session). Mais il **capture l'intention** : *quiconque accepte d'installer de l'outillage pour coordonner des agents ouvre d'abord le couteau suisse à 61 k★*. Tu te bats pour les miettes : ceux qui veulent coordonner **sans rien installer de lourd** — c'est‑à‑dire le régulé — qui ne t'achète pas. Le pari n'est pas tué techniquement ; il est **encerclé commercialement**.

---

## 4. La démo « wow » est‑elle un vrai wow ou un gadget ?

**Gadget compétent déguisé en wow.** Trois clous : ⚠️

1. **Elle est DRIVÉE** — tu l'admets (§29) : nudges clavier, autonomie non réelle. Ce que voit le spectateur, c'est : *« j'ai tapé un message dans le pane A, il apparaît dans le pane B »*. C'est… ce que fait n'importe quel chat. Le cadre « trois éditeurs rivaux » est malin, mais l'action sous‑jacente est de la **messagerie banale sur tmux** — exactement le « scripts + cérémonie » que tu redoutes.
2. **Le money‑frame chiffrement est du théâtre.** Montrer `csend journal` → `sha256:… (jamais le clair)` **sur localhost**, entre des process que le **même utilisateur** possède, c'est un wow d'**ingénieur sécu**, pas de dev — et tu as **déjà concédé** que la crypto est un passif en local. Pire : ça **invite** le commentaire HN cynique *« pourquoi du E2E entre mes propres terminaux ? »*. Tu mets en scène ta propre objection.
3. **Le seul beat réellement neuf** — la session Claude vivante qui se corrige sur le verdict d'un **autre vendor** sans copier‑coller — dure **~2 secondes, enfoui au milieu**, et c'est la **seule** partie que 50 lignes de bash ne reproduisent pas. Ton propre test de survie dit : « < 90 s pour voir ce que bash ne fait pas ». Réponse honnête : **pendant ~88 s sur 90, bash le fait**. Le wow, c'est 2 secondes ; le gadget, c'est le reste.

Le vrai wow serait : agent A en plein travail commet une erreur subtile, agent B (autre vendor) la **rattrape de façon autonome, non sollicitée**, et A **se corrige visiblement**. Tu écris toi‑même que tu **ne peux pas** le montrer aujourd'hui (cadence non calibrée). Donc : **ta démo ne peut pas montrer le vrai wow.** Tu filmes la cérémonie autour du moment magique, pas le moment magique.

---

## 5. Verdict tranché

### NO‑GO en tant que produit/société. GO pour un WEEK‑END, pas 6 semaines.

**Pourquoi NO‑GO entreprise :** c'est structurellement **une feature en quête d'un propriétaire**, bâtie sur les implémentations de hooks de **trois rivaux**, dont **chacun peut te casser, te sandboxer ou t'absorber** (ton propre G4). Les trois propriétaires naturels de ce wedge sont les trois CLIs eux‑mêmes. Une société ne se construit pas sur un mince ruban entre trois plateformes qui peuvent le supprimer dans une release. Ajoute : aucun payeur prouvé (Q1), use‑case vitamin (Q2), intention déjà captée par un incumbent à 61 k★ (Q3), démo dont le wow réel n'est pas montrable (Q4).

**Ce qui mérite de survivre :** **Pivot B**, et lui seul — *« inject & receive into any running agent CLI »* comme **petite lib Go autonome**, livrant le `/second-opinion` cross‑vendor live‑in‑session. **Jette le bus, la crypto, le registre, le web‑of‑trust, la marque, le treadmill CI, l'ICP SecOps.** C'est de la soustraction (§57) : il reste **le seul primitif neuf**.

**Tes kill‑gates sont excellents (rare !) — mais tu as déjà écrit ton propre arrêt de mort :** G6 (« 0/15 interviews citent chiffrement/souveraineté → le moat est un passif ») est **quasi certain de se déclencher**, puisque *tu affirmes toi‑même* que la crypto est un passif. Tu connais déjà le résultat. N'attends pas J+28 pour acter ce que tu sais à J+0.

### Le premier pas EXACT (et il n'est PAS « Lun‑Mar, P0 »)

Ton plan est à l'envers : infra‑first, démo‑jeudi. **Inverse‑le. Démo‑first, infra‑jamais‑si‑ça‑rate.**

> **Jour 1 — un seul spike, une seule question :** sur les hooks **d'aujourd'hui**, un verdict produit par Codex/Gemini **retombe‑t‑il dans la session Claude vivante et change‑t‑il visiblement sa prochaine action**, avec une latence acceptable, capturé dans **un screen‑recording de 8 secondes** ?

- Si **oui** et que ce moment de 2 s est **indéniable** à l'écran → tu as le seul actif qui compte. Mets le GIF en haut d'un gist, balance‑le sur HN/r/LocalLLaMA **tel quel**, et laisse G2 (J+14) trancher. *Ensuite* seulement tu décides si les 6 semaines valent le coup.
- Si **non / latence pourrie / il faut tricher au clavier pour que ça marche** → **NO‑GO immédiat**, tu as économisé 6 semaines pour le prix d'une journée.

Tout le reste de ta semaine 1 (P1 installateurs, P2 enveloppe, P3 async, marque, rebrand, kill‑gates doc) est **du polish sur un bateau dont tu n'as pas prouvé qu'il flotte**. Le rebrand « csend » : vrai problème, **mauvais moment** — on ne débat pas du nom de la coque avant de savoir si elle prend l'eau. Defer.

**La règle dure, version impitoyable :** si le screen‑recording de 8 secondes du Jour 1 ne fait pas dire à un inconnu *« attends, comment ça il se corrige tout seul sur l'avis d'un autre modèle ? »* — alors csend **est** « scripts + cérémonie », le marché minuscule se referme, et la seule bonne décision est de ne pas écrire la ligne 2.
