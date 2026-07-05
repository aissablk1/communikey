# communikey — Vision complète (développée)

> Développé à partir de la demande d'Aïssa du 2026-07-03, croisée avec l'état réel du
> code (audit du même jour, cf. `docs/NEXT.md`) et la liste des CLI hooks natifs de
> cmux/Vibe Island (capture d'écran, Réglages → Intégrations, 2026-07-03 23:17). Chaque
> case « fait » a été vérifiée par lecture de code ou exécution réelle, pas supposée
> (§29). Ce document ne remplace pas `docs/NEXT.md` (qui reste la liste honnête et
> datée de ce qui manque) — il en est la version développée : le **pourquoi**, la
> **carte complète**, et les idées non demandées mais pertinentes.

---

## 0. La phrase (positionnement, en une fois)

Tu pilotes plusieurs agents de code en parallèle — une fenêtre par tâche, parfois
plusieurs machines — et il te manque le câble entre eux : aucune API officielle
n'injecte un message dans une session d'agent CLI vivante, alors chaque coordination se
fait à la main, par copier-coller, ou en confiant tes prompts à un relais qui les lit en
clair. Une session doit pouvoir en prévenir une autre, la relancer, lui demander une
seconde opinion — en lisant d'abord son état (idle, occupée, en confirmation) pour ne
jamais lui couper la parole — à travers les terminaux, les éditeurs et les machines,
sans jamais exposer le contenu de tes prompts à qui que ce soit d'autre que le
destinataire. Chaque message y est signé par une identité Ed25519 — tu sais qui parle,
pas seulement qu'un secret partagé est bon — et chiffré pour son destinataire en
hybride post-quantique (X25519 ⊕ ML-KEM-768), avec une recovery souveraine par parts
Shamir et phrase BIP-39 si jamais tu perds l'accès. Alors, pour faire court,
« communikey » est : le bus de messages chiffré et signé qui fait enfin parler tes
sessions d'agents de code entre elles — souverain, auditable, sans dépendre d'un
orchestrateur propriétaire.

## 1. Le besoin, reformulé en une carte

Ta demande couvre cinq axes. Voici où chacun en est **réellement** (pas la doc, le code) :

| Axe | Demandé | État réel (audit 2026-07-03) |
|---|---|---|
| Sessions Claude Code entre elles | ✅ | **Fait** — cœur du bus, `bus.go`/`cmux.go` |
| Relations familiales (père/enfants) « ou non » | ✅ | **Fait** — `relations.go` : `link`/`unlink`/`tree`, arêtes optionnelles, anti-cycle vérifié |
| Autres providers (Codex, Gemini, « et autre ») | ✅ | **Partiel** — 3/9+ providers connus (voir §2) |
| Agents locaux, Agent Teams, sous-agents, « armées » | ✅ | **Partiel** — flotte visible (`agents`/`whoami`/`register`), **bridge Agent Teams non fait** (format mailbox inconnu) |
| Cross-CLI / cross-terminal / cross-OS / cross-workspace | ✅ | **Fait pour l'essentiel** — voir §5 (Windows = coop-only, limite physique assumée) |
| CLI dédiée richement configurable, variantes d'aide | ✅ | **Fait** — les 7 variantes (`h -h --h help -help --help -?`) vérifiées identiques au byte près |

Le contrat implicite dans ta phrase — « même ce que je n'y ai pas pensé » — est traité
en §7.

---

## 2. Cartographie des providers — la vraie liste (capture cmux 2026-07-03)

Ta demande citait « Codex, Gemini CLI, et autre ». Le panneau **Intégrations → CLI
Hooks** de cmux/Vibe Island (que tu pilotes déjà) référence **9 CLI d'agents nommés**,
plus un mécanisme d'ajout libre. C'est la liste de référence à viser, pas une
supposition :

| Provider (cmux) | Statut dans communikey | Ce qu'il faut pour le faire |
|---|---|---|
| **Claude Code** | ✅ Calibré sur vrais écrans (`state.go`), battle-tested | — |
| **Codex** | 🟡 Adaptateur provisoire (`adapters.go`), patterns vérifiés sur le repo officiel (tag `rust-v0.142.3`) mais jamais validés sur écran *live* | Capturer un vrai écran Codex en confirm/busy/idle |
| **Gemini CLI** | 🟡 Idem, vérifié sur le bundle installé `@google/gemini-cli 0.40.1` | Capturer un vrai écran Gemini live |
| **OpenCode** | ⚪ Absent | Point d'extension `Provider` prêt — capturer des écrans réels d'abord |
| **Cursor Agent** | ⚪ Absent | idem |
| **Droid** (Factory AI) | ⚪ Absent | idem |
| **Hermes** | ⚪ Absent — **à clarifier** : nom partagé avec ton pont de messagerie MCP `hermes` (Telegram/Discord/Slack/WhatsApp/Signal/Matrix) ; vérifier si cmux référence le même outil ou un agent CLI distinct avant de calibrer quoi que ce soit |
| **Pi Agent** | ⚪ Absent, connaissance limitée de ce CLI précis | idem — ne pas deviner ses patterns |
| **Kiro CLI** (AWS) | ⚪ Absent | idem |
| **« Ajouter une branche CLI… »** | — | cmux a déjà un mécanisme d'ajout libre côté hook ; communikey devrait avoir l'équivalent côté détection d'état (§3) |

**Règle inchangée (§2/§29)** : chaque nouveau provider suit la même discipline que
Codex/Gemini — patterns calibrés sur de **vrais écrans capturés**, jamais des fixtures
inventées. Le risque d'un mauvais calibrage n'est pas cosmétique : un faux idle
soumettrait un message dans une session qui n'est pas prête. La conception
« confirm > busy > idle-à-double-preuve > unknown » (déjà en place) rend l'échec
**toujours sûr** (on rate un message plutôt que d'en soumettre un au mauvais moment) —
donc ajouter des providers ne dégrade jamais la sûreté des providers existants.

## 3. Rendre l'ajout de provider scalable (le vrai levier, au-delà de « ajoute Kiro »)

Aujourd'hui, un provider est un `patternProvider{...}` **codé en dur dans
`adapters.go`** — ajouter un CLI exige un rebuild Go. Avec 6 providers restants (et
d'autres à venir — cmux propose déjà « Ajouter une branche CLI… », donc la liste
grandira), coder chaque adaptateur à la main ne passe pas à l'échelle.

**Proposition** (le point que tu n'as pas demandé mais qui découle directement de ta
liste) : externaliser les patterns dans un fichier de config (`providers.yaml` ou
`.json`, chargé au démarrage, avec un jeu par défaut embarqué pour ne rien casser).
Effets :
- Ajouter un provider devient éditer un fichier, pas recompiler — aligné avec le
  modèle « Ajouter une branche CLI… » que cmux propose déjà côté hooks.
- Une nouvelle commande **`communikey provider test <name>`** deviendrait possible :
  l'utilisateur colle un écran réel de SON CLI, communikey applique le pattern proposé
  et affiche l'état détecté — ce qui transforme le blocage actuel (« il faut de vraies
  captures ») en **boucle de calibration communautaire** au lieu d'un goulot
  d'étranglement qui repose uniquement sur toi.
- `communikey provider list` afficherait l'état de calibration de chaque provider
  (calibré / provisoire / absent) — la transparence de `docs/NEXT.md`, mais dans la CLI
  elle-même.

## 4. Flottes, familles, teams, sous-agents — « des armées de sessions »

- **Familles (père/enfants)** : fait (`link`/`unlink`/`tree`), arêtes **logiques**
  (déclarées), pas spatiales — exactement ce qui permet le « ou non » de ta demande
  (une session peut rester orpheline, c'est un état valide).
- **Flotte visible tous providers/OS/terminaux** : `communikey agents` fait déjà ça —
  c'est la commande qui répond à « armées de sessions de providers et d'agents ».
- **Sous-agents** (agents que TU lances depuis une session, comme les subagents de
  Claude Code) : pas un concept séparé à construire — un sous-agent qui s'enregistre
  (`communikey register`) apparaît comme un nœud comme un autre, et `link` peut en
  faire un enfant de la session qui l'a lancé. Le modèle couvre déjà ce cas ; ce qui
  manque est l'**auto-link au spawn** (aujourd'hui manuel) — amélioration possible, pas
  bloquante.
- **Agent Teams** (la mailbox propriétaire `~/.claude/teams/…`) : seul vrai bloqué de
  cette section, faute du format réel (non documenté publiquement, §2/§29 — pas de
  format deviné).

## 5. Cross-plateforme — CLI, terminal, OS, workspace

| Dimension | État réel |
|---|---|
| Cross-CLI (bash, zsh, sh, PowerShell…) | Binaire Go compilé, zéro dépendance — **agnostique du shell appelant** par construction |
| cmux, tmux | Backends dédiés livrés (`cmux.go` + backend tmux), tests verts |
| Terminal.app / autres émulateurs | Couvert par la voie **coopérative** (inbox + hook), universelle par conception — seule l'**injection clavier live** est liée à cmux/tmux |
| Linux (amd64/arm64) | ✅ build vérifié par compilation croisée réelle |
| macOS (arm64) | ✅ idem |
| Windows | ✅ build réel, **coop-only assumé** : l'injection clavier est physiquement impossible (pas de PTY maître partageable, ConPTY ne s'attache pas à un process déjà lancé) — pas un manque, une contrainte du système d'exploitation |
| Chromebook | À clarifier : si c'est via **Crostini (conteneur Linux)**, le build `linux/amd64` ou `linux/arm64` couvre déjà le cas ; si c'est de la ChromeOS native (extension/app Android), c'est un runtime entièrement différent, hors périmètre CLI |
| Cross-workspace | Fait — `communikey list`/`agents` traversent tous les workspaces visibles, sans bascule |

## 6. La CLI `communikey` elle-même — déjà riche, quelques ajouts qui la complètent

**Déjà fait** (vérifié par exécution le 2026-07-03) : ~20 sous-commandes, 7 variantes
d'aide identiques au byte près (`h -h --h help -help --help -?`), sortie JSON sur
`journal`.

**Propositions (ce que tu n'as pas listé, mais qui suit naturellement)** :
- **Complétion shell** (bash/zsh/fish) — attendu de toute CLI mature, coût faible.
- **`communikey doctor`** — diagnostic en une commande : hooks installés ? PATH
  correct ? vault présent et lisible ? contacts enregistrés ? Répond à « pourquoi ça ne
  marche pas » sans fouiller la doc.
- **Page de man** (`communikey man` ou fichier `.1` généré au build).
- **Fichier de config persistant** (`~/.communikey/config.toml`) en complément des
  variables d'env actuelles (`COMKEY_*`) — pour les réglages qu'on ne veut pas
  ré-exporter à chaque session.
- **`--json` étendu** à `list`/`tree`/`agents` (déjà présent sur `journal`) — pour que
  d'autres outils/scripts consomment l'état sans parser du texte tabulaire.

## 7. Ce que tu n'y as pas pensé — au-delà de la CLI

- **Boucle de calibration communautaire** (§3) : transforme le vrai goulot
  d'étranglement (calibrer 6 providers) en quelque chose que la communauté peut
  alimenter, plutôt qu'un travail que toi seul peux faire.
- **Mode démon** (`communikey serve` en service `systemd`/`launchd`) pour le réseau
  multi-machine, plutôt qu'un process lancé à la main à chaque fois.
- **Export d'audit** : `communikey journal` existe déjà (hash only) — un export signé
  horodaté (pour preuve d'activité, sans exposer le clair) prolongerait naturellement
  la posture « auditable » déjà revendiquée.

## 8. Priorisation proposée (impact/coût, pas un ordre imposé)

1. **Clarifier "Hermes"** dans le panneau cmux (même outil que ton MCP, ou CLI distinct ?) — 5 minutes, lève une ambiguïté avant de calibrer quoi que ce soit.
2. **Externaliser les patterns provider** (§3) — le levier qui rend TOUT le reste (6 providers restants + les futurs) moins coûteux à ajouter.
3. **Calibrer Codex/Gemini sur écrans réels** — les deux adaptateurs provisoires existants, complets à 90 %, qui n'attendent qu'un test en conditions live.
4. **`communikey doctor`** — gain UX rapide, indépendant du reste.
5. **Bridge Agent Teams** — bloqué tant que le format mailbox n'est pas inspecté sur une vraie session.
6. **Nouveaux providers (OpenCode, Cursor Agent, Droid, Pi Agent, Kiro CLI)** — une fois #2 fait, chacun devient un fichier de pattern à calibrer, pas un rebuild Go.

## 9. Rappel honnête (ne pas dupliquer `docs/NEXT.md`)

Les blocages « faute de vraies données » (captures d'écran Codex/Gemini/nouveaux
providers, format mailbox Agent Teams) restent exactement ceux documentés dans
`docs/NEXT.md` — ce document n'invente rien de nouveau sur ce point, il les remet dans
le cadre de la vision complète et les priorise (§8).

---

**Auteur** : Aïssa BELKOUSSA
