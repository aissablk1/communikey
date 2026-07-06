---
title: "communikey — Vision maximale (développée, 2026-07-05)"
date: 2026-07-05
auteur: Aïssa BELKOUSSA
projet: communikey
version: 2.0
statut: validé
tags: [stratégie, vision, positionnement, providers, sécurité, cli]
---

# communikey — Vision maximale (développée)

> Ce document fusionne trois sources qui vivaient séparément : la vision fonctionnelle du
> 07-03 (`communikey-vision-complete.md`, les cinq axes), le teardown concurrentiel du 07-01
> (`communikey-axes-de-depassement.md`, hcom) et l’état crypto du 03/07 encore non publié
> (CHANGELOG « vers 0.3.0 »). À cela s’ajoute une relecture directe du code, faite pour cette
> version — `main.go`, `relations.go`, `provider.go`, `adapters.go`, `hook.go` — pour vérifier
> chaque affirmation plutôt que la recopier d’un autre document. Rien n’est remplacé : les
> trois fichiers d’origine restent lisibles comme trace datée du raisonnement ; celui-ci en
> est la synthèse à jour, plus complète.

## Vérification du pitch

Le texte court que tu as validé — « tu pilotes plusieurs agents de code en parallèle […]
alors, pour faire court, « communikey » est : le bus de messages chiffré et signé […] » —
tient entièrement à la relecture du code, pas seulement des docs :

- **Signature Ed25519** : `crypto.go` signe l’enveloppe scellée avec `crypto/ed25519` (stdlib
  Go, aucune dépendance).
- **Hybride post-quantique X25519 ⊕ ML-KEM-768** : confirmé dans `crypto.go` — encapsulation
  ML-KEM-768 (`crypto/mlkem`, FIPS 203) vers la clé statique du destinataire, échange X25519
  éphémère par message (`crypto/ecdh`), les deux secrets combinés par HKDF-SHA256.
- **Recovery Shamir + BIP-39** : `shamir.go` (seuil K-sur-N, corps GF(2⁸), from-scratch, testé)
  et `bip39.go` (24 mots, wordlist anglaise officielle) sont réels, pas des placeholders.
- **Lecture d’état avant action** : `provider.go`/`adapters.go` imposent l’ordre
  confirm → busy → idle (double signal) → unknown à *chaque* provider enregistré — jamais
  d’Entrée envoyée à l’aveugle.
- **Les trois éditeurs cités** (Claude Code, Codex, Gemini) sont les trois seuls providers
  réellement implémentés aujourd’hui, pas une liste aspirationnelle.

Le texte n’a donc besoin d’aucune correction. Ce qui suit développe, autour de lui, les six
axes que tu as listés — plus ce qu’ils font apparaître une fois qu’on lit le code jusqu’au
bout.

## Le trou qu’il comble

Il n’existe **aucune API officielle** pour injecter un message dans une session d’agent CLI
déjà vivante — ce n’est pas une supposition, ce sont deux demandes de fonctionnalité réelles
sur le dépôt Claude Code, l’une fermée l’autre ouverte
([#24947](https://github.com/anthropics/claude-code/issues/24947),
[#27441](https://github.com/anthropics/claude-code/issues/27441)). En l’absence de ce câble,
la coordination entre sessions se fait à la main : copier-coller entre fenêtres, ou passer par
un relais qui lit les messages en clair.

communikey répond par deux chemins qui ne s’excluent pas :

- une **voie coopérative** — inbox par fichiers, `register`/`inbox`/`recv` — qui marche sur
  tout OS et tout provider, sans multiplexeur ni démon ; c’est la colonne vertébrale
  universelle ;
- une **injection clavier live**, réservée à cmux et tmux (Unix), qui pilote une session sans
  attendre qu’elle relève son inbox — un repli pour les interfaces texte qui n’ont pas encore
  de hook de réception.

Une seule adresse (un nom de workspace, un session-id ou une ref `surface:N`), un seul outil,
et le message trouve toujours un chemin.

## Carte des six axes demandés

| Axe demandé | État vérifié le 2026-07-05 |
|---|---|
| Invocable par chaque session, dans les deux sens | **Fait** — `register`/`inbox`/`recv` (coopératif) + `send`/`hook`/`watch` (live), skill Claude Code `~/.claude/skills/communikey` |
| Relations familiales, « ou non » | **Fait** — `relations.go` : arêtes déclarées, optionnelles, anti-cycle |
| Autres providers (Codex, Gemini, « et autre ») | **Partiel** — 3 calibrés sur 9+ providers connus, mais l’extension est désormais **sans recompilation** (`providers.json` + `provider list`/`test`, livré le 07-05) |
| Agents locaux, teams, sous-agents, « armées » | **Partiel** — flotte visible, sous-agents modélisés ; **découverte** Agent Teams livrée le 07-05 (`communikey teams`, schéma réel capturé) ; la **livraison** dans la mailbox reste à construire (format non observé) |
| Cross-CLI, cross-terminal, cross-OS, cross-workspace | **Fait pour l’essentiel** — Windows et Chromebook ont des nuances réelles, détaillées plus bas |
| CLI dédiée, richement configurable, variantes d’aide | **Fait** — ~20 sous-commandes, 7 variantes d’aide identiques au byte près |

Chaque ligne est développée dans sa propre section ci-dessous.

## Invocabilité, dans les deux sens

Une session rejoint le bus avec `communikey register` — n’importe quel terminal, n’importe
quel provider, n’importe quel OS — et devient visible dans `communikey agents`. À partir de
là, deux verbes suffisent : `inbox` pour déposer un message dans la boîte d’une autre session,
`recv` pour relever la sienne. Rien n’exige cmux ou tmux pour cette voie ; c’est le socle qui
marche partout.

Là où un multiplexeur est disponible, `send` fait mieux qu’un simple dépôt : il **lit l’écran**
de la cible avant d’écrire (`read`, `key`), et ne valide (appuie sur Entrée) que si l’état
détecté est idle. Les garde-fous sont dans le code, pas dans la documentation :
`communikey` refuse de s’auto-injecter dans la session courante, et refuse d’envoyer une
touche Entrée à une session non reconnue ou face à un prompt de confirmation — sauf
`--force`, explicite et assumé. C’est la propriété que le pitch résume par « ne jamais lui
couper la parole ».

La réception, elle, se fait sans polling grâce à `communikey hook`, câblé comme hook natif de
chaque éditeur — `UserPromptSubmit` pour Claude Code et Codex, `BeforeAgent` pour Gemini
(`hook.go`). À chaque tour, le hook draine l’inbox et injecte les messages reçus dans le
contexte de l’agent, sous la forme exacte que chaque CLI attend (`hookSpecificOutput` pour
Claude/Codex, stdout brut pour Gemini) — sans bruit si l’inbox est vide. Pour un humain qui
veut suivre en direct sans hook, `communikey watch` fait le tail live.

Et communikey s’invoque aussi **en langage naturel** : le README l’expose comme skill Claude
Code (`~/.claude/skills/communikey`), et cette même skill figure aujourd’hui parmi celles
chargées dans tes sessions — « dis à SACEM de relancer le build » se traduit en un appel
`communikey inbox`/`send` sans que tu aies à taper la commande toi-même.

## Relations familiales, y compris l’absence de lien

`relations.go` modélise une famille de sessions par des arêtes **logiques**, pas spatiales :
le parent d’une session est celle qui l’a lancée ou la possède, une information que la
disposition des panneaux cmux n’encode pas. `communikey link <enfant> <parent>` déclare le
lien, `unlink` le retire, `tree` dessine le graphe en respectant l’ordre des enfants et en
affichant aussi les nœuds hors-ligne connus par leur nom. Un enfant n’a jamais qu’un seul
parent ; en créer un second remplace silencieusement l’ancien lien plutôt que d’empiler des
arêtes contradictoires. Toute tentative de lien qui créerait un cycle est refusée
(`wouldCycle` remonte l’arbre jusqu’à la racine ou jusqu’à 1000 sauts, au cas où).

Le « ou non » de ta demande est pris au sérieux dans le modèle lui-même : une session sans
parent ni enfant est un **état valide**, pas une erreur ni un cas particulier à gérer à part —
`tree` la montre simplement comme une racine isolée. À partir des liens déclarés, `send
--down` diffuse aux enfants directs, `--up` remonte au parent, `--to-siblings` touche les
sessions qui partagent le même parent, `--to-descendants` couvre tout le sous-arbre — et
`--from <session>` permet de raisonner depuis une base différente de la session courante.

Un sous-agent que tu lances depuis une session (l’équivalent des subagents Claude Code) n’est
pas un concept séparé à construire : dès qu’il s’enregistre lui-même, il apparaît comme un
nœud comme un autre, et un `link` explicite peut en faire un enfant de la session qui l’a
lancé. Ce qui manque encore, c’est l’auto-link au moment du spawn — aujourd’hui manuel, une
amélioration, pas un blocage.

## Interopérabilité multi-providers

Le point d’extension existe déjà : un `Provider` n’a que deux méthodes, `Name()` et
`Detect(screen string) State` (`provider.go`), et le registre essaie chaque provider dans
l’ordre jusqu’au premier qui reconnaît l’écran. Claude reste **toujours en tête** de ce
registre — son détecteur est réglé à la main sur de vrais écrans capturés, battle-tested — et
les autres n’interviennent que quand Claude s’abstient.

Trois providers existent aujourd’hui :

| Provider | Statut | Base de calibrage |
|---|---|---|
| Claude Code | **Fait**, éprouvé | vrais écrans capturés |
| Codex | Adaptateur provisoire | dépôt officiel `openai/codex`, tag `rust-v0.142.3` — jamais confirmé sur écran *live* |
| Gemini CLI | Adaptateur provisoire | bundle réellement installé `@google/gemini-cli 0.40.1` — jamais confirmé sur écran *live* |

Codex et Gemini partagent avec Claude la même discipline de sûreté : un dialogue de
confirmation n’est **jamais** lu comme idle, et l’idle exige une **double preuve** — une boîte
de saisie vide *et* un pied de page distinctif du provider (`adapters.go`). C’est ce double
signal qui empêche un shell nu, sans agent dedans, d’être pris pour une invite soumissible —
le seul faux positif vraiment dangereux. Concrètement, l’état « busy » de Codex partage le
texte « esc to interrupt » avec Claude : un écran Codex occupé peut être attribué au nom
« claude » (Claude passe en premier), mais l’**état détecté reste correct** — seul le nom du
provider peut différer, jamais la sûreté de la livraison.

Six providers restent absents, listés d’après le panneau « Intégrations → CLI Hooks » de
cmux/Vibe Island (la même capture que celle du 07-03, revérifiée avant d’écrire ce document) :
**OpenCode**, **Cursor Agent**, **Droid** (Factory AI), **Hermes**, **Pi Agent**, **Kiro CLI**
(AWS) — plus un mécanisme d’ajout libre côté cmux, ce qui veut dire que la liste grandira.
« Hermes » est clarifié depuis (recherche web du 2026-07-05, source primaire) : c’est très
vraisemblablement le **même produit** que ton pont MCP de messagerie déjà installé, pas un
homonyme distinct. [Hermes Agent](https://github.com/NousResearch/hermes-agent) (Nous Research,
sorti en février 2026) est un agent autonome open source avec **sa propre CLI/TUI** (édition
multiligne, autocomplétion de slash-commands, sortie d’outils en streaming — la même famille
d’interaction que Claude Code/Codex/Gemini) et une **passerelle multi-plateforme unique** :
Telegram, Discord, Slack, WhatsApp, Signal, CLI — cinq des six canaux cités dans la description
de ton MCP `hermes` (« Telegram, Discord, Slack, WhatsApp, Signal, Matrix ») correspondent
exactement. Support Windows natif, mémoire persistante inter-session, boucle d’apprentissage,
40+ outils, agnostique au modèle. Si l’hypothèse tient (à confirmer visuellement sur le panneau
cmux — non revérifié ce tour-ci, capture inchangée d’après ta réponse), « Hermes » est un **vrai
7ᵉ candidat provider** légitime — une CLI d’agent réelle, pas une simple passerelle de
notifications — mais son calibrage suivrait la même règle que Codex/Gemini : patterns dérivés de
sa [documentation CLI réelle](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/reference/cli-commands.md),
jamais inventés, puis validation sur écran live.

**Le vrai levier n’était pas d’ajouter Kiro à la main — c’est fait, livré le 07-05.** Un
provider était un `patternProvider{...}` codé en dur dans `adapters.go` : ajouter un CLI
exigeait de recompiler. Avec six providers restants — et une liste qui s’allonge côté cmux —
coder chaque adaptateur un par un ne tenait pas à l’échelle. `~/.claude/communikey/providers.json`
(purement additif : claude/codex/gemini restent compilés en dur, inchangés, zéro régression)
change maintenant la nature du problème : ajouter un provider est éditer un fichier JSON — pas
YAML, le projet reste zéro-dépendance — plutôt que recompiler. `communikey provider test <name>`
colle un écran réel sur stdin et affiche l’état détecté ; `communikey provider list` affiche le
calibrage de chacun (calibré/provisoire/personnalisé/absent) directement dans la CLI, sans
ouvrir `docs/NEXT.md`. Le calibrage devient une boucle que la communauté peut alimenter, pas un
goulot d’étranglement que toi seul peux lever.

Le test de bout en bout (pas seulement les tests unitaires) a immédiatement révélé un vrai
piège à documenter : un `idle_prompt`/`idle_footer` sans le flag `(?m)` ne matche jamais un
écran multi-lignes (`^`/`$` bornent alors toute la chaîne, pas chaque ligne) — exactement le
genre d’erreur qu’un futur contributeur ferait. `provider list` le rappelle maintenant
explicitement.

Un détail que la lecture du code fait apparaître et que personne n’a demandé : l’ajout d’un
provider a en réalité **deux** coûts, pas un seul. `provider.go`/`adapters.go` couvrent la
détection d’état ; mais `hook.go` a son propre aiguillage pour le *snippet d’installation* du
hook (`hookInstallFor`), et son `switch` ne connaît que `"codex"` et `"gemini"` — tout autre
nom de provider retombe silencieusement sur le snippet pensé pour Claude. Le jour où un
`OpenCode` ou un `Cursor Agent` est ajouté côté détection, il faudra aussi lui donner sa
propre branche dans `hookInstallFor`, sans quoi `communikey hook --install opencode`
afficherait un snippet Claude qui ne correspond à rien chez OpenCode.

## Agents, teams, sous-agents : les armées de sessions

`communikey agents` répond directement à « des armées de sessions de providers et d’agents » —
la flotte coopérative visible tous providers, tous OS, tous terminaux confondus, sans
distinction entre une session cmux, une session tmux ou une session qui n’a rejoint le bus que
par `register`. Combinée au graphe familial de la section précédente, elle donne une vue
complète : qui existe, qui est enfant de qui, et qui n’a pas de lien du tout.

Le seul point réellement bloqué de cette section est le pont vers **Agent Teams**, précisé le
07-05 sur la [doc officielle](https://code.claude.com/docs/en/agent-teams). Fait établi, pas
supposé : c'est une fonctionnalité **expérimentale, désactivée par défaut**
(`CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1` dans `settings.json` pour l'activer). Un lead spawn
des teammates (chacun une session Claude Code séparée) qui se coordonnent via une **liste de
tâches partagée** et une **mailbox** de peer-to-peer (outil interne `SendMessage`). Deux
répertoires, un nom dérivé de la session (`session-` + 8 premiers caractères du session-id) :
`~/.claude/teams/{team-name}/config.json` (config d'équipe — supprimé à la fin de la session)
et `~/.claude/tasks/{team-name}/` (liste de tâches — persiste, jamais uploadée). Le config
contient un tableau `members` avec le nom, l'agent-id et le type de chaque teammate — **« lisible
pour découvrir les autres membres », mais explicitement « ne pas éditer à la main »** (regénéré
à chaque mise à jour d'état). Il existe aussi trois hooks dédiés (`TeammateIdle`, `TaskCreated`,
`TaskCompleted`) — mais ce sont des hooks de **garde** (sortie code 2 = bloquer + feedback), pas
des hooks d'injection de contexte comme `UserPromptSubmit` : un mécanisme différent de celui
que `hook.go` sait déjà câbler.

**Débloqué en partie le 07-05** : le flag a été activé dans `~/.claude/settings.json`, et un
vrai `config.json` a été capturé en forçant un pseudo-tty (le mode `-p` headless ne spawne pas
de vraies teammates — il les *simule* dans sa réponse texte, sans jamais toucher
`~/.claude/teams/`). Schéma confirmé, camelCase, epoch millis :

```json
{
  "name": "session-2a1598bb", "createdAt": 1783258279972,
  "leadAgentId": "team-lead@session-2a1598bb",
  "leadSessionId": "2a1598bb-c89b-4929-a3b9-ce8b85b4a482",
  "members": [{
    "agentId": "team-lead@session-2a1598bb", "name": "team-lead",
    "agentType": "team-lead", "joinedAt": 1783258279972,
    "tmuxPaneId": "leader", "cwd": "…", "subscriptions": [], "backendType": "in-process"
  }]
}
```

`agentteams.go` lit ce fichier (jamais ne l'écrit) et `communikey teams` liste les équipes et
membres détectés — la **découverte**, exactement comme `communikey agents` pour la flotte
coopérative. Ce qui reste honnêtement incomplet : la session de capture s'est bloquée sur le
dialogue « fais-tu confiance à ce dossier ? » avant qu'un vrai *teammate* (pas seulement le
lead) ne rejoigne — le tableau `members` n'a donc été vu qu'avec une seule entrée. La forme est
la même quel que soit le nombre de membres, donc le parseur reste correct, mais un
**deuxième membre réel n'a jamais été observé**. Plus important : le format de la **mailbox**
(l'outil interne `SendMessage`) reste totalement non observé — router.go réserve la voie
(`ChannelBridge`, `TargetInfo.Bridge`) mais rien n'écrit encore dedans, faute d'un vrai message
à inspecter. La **livraison** dans une Agent Team reste donc à construire ; seule la
**découverte** est faite.

## Cross-CLI, cross-terminal, cross-OS, cross-workspace

communikey est un binaire Go compilé, sans dépendance : il est **agnostique du shell
appelant** par construction — bash, zsh, sh, PowerShell n’ont rien de spécifique à gérer,
puisqu’il n’y a rien d’autre à faire qu’exécuter un binaire.

| Dimension | État vérifié |
|---|---|
| cmux, tmux | Backends dédiés livrés, interchangeables derrière une interface `Backend`, tests verts |
| Terminal.app et autres émulateurs | Couverts par la voie coopérative, universelle par conception — seule l’injection clavier live reste liée à cmux/tmux |
| Linux (amd64/arm64) | Build vérifié par compilation croisée réelle |
| macOS (arm64) | Build vérifié |
| Windows | Build réel, **coop-only** : l’injection clavier est physiquement impossible sur cette plateforme — pas de PTY maître partageable, et ConPTY ne s’attache pas à un process déjà lancé. Ce n’est pas un manque, c’est une contrainte du système d’exploitation. |
| Chromebook | À clarifier selon le runtime réel : sous **Crostini** (conteneur Linux), le build `linux/amd64` ou `linux/arm64` couvre déjà le cas ; en ChromeOS natif (extension ou app Android), c’est un runtime entièrement différent, hors du périmètre d’un binaire CLI |
| Cross-workspace | `communikey list`/`agents` traversent tous les workspaces visibles sans bascule manuelle |

Les clients **mobiles** (iOS/iPadOS, Android) suivent une logique à part : le sandbox de ces
plateformes empêche toute injection clavier, donc ils ne rejoignent le bus que comme
**clients** — voir l’état des sessions, approuver, diffuser un message — à la façon d’un
`omnara`. C’est une phase à part de la feuille de route (phase 4), pas encore livrée.

## La CLI elle-même

`main.go` expose aujourd’hui une vingtaine de sous-commandes : `list`/`ls`, `tree`, `link`,
`unlink`, `send` (avec `--stage`/`--send`/`--force`/`--up`/`--down`/`--to-siblings`/
`--to-descendants`/`--from`), `read`, `key`, `register`, `agents`, `whoami`, `recv`, `inbox`,
`id` (`--create`/`--export`), `contact` (`add`/`list`), `recovery` (`split`/`combine`/
`phrase`/`from-phrase`), `serve` (`--addr`/`--tls`), `remote` (`--tls`/`--pin`), `hook`
(`--install`/`--provider`), `watch` (`--interval`), `journal` (`--json`), `version`, et
un diagnostic caché `_why`.

Les sept variantes d’aide demandées — `h`, `-h`, `--h`, `help`, `-help`, `--help`, `-?` —
retombent toutes dans la même branche du `switch` de `main.go` et impriment le même texte
d’usage au byte près ; ce n’est pas un ajout de surface, c’est déjà le comportement réel du
binaire.

Ce qui n’est pas encore fait, mais suit naturellement de ce qui existe déjà :

- **Complétion shell** (bash, zsh, fish) — attendu de toute CLI mature, coût faible vu qu’il
  n’y a qu’un seul binaire à décrire.
- **`communikey doctor`** — un diagnostic en une commande : hooks installés ? PATH correct ?
  vault présent et lisible ? contacts enregistrés ? Répond à « pourquoi ça ne marche pas »
  sans avoir à ouvrir la documentation.
- **Page de manuel** (`communikey man`, ou un fichier `.1` généré au build).
- **Fichier de configuration persistant** (`~/.communikey/config.toml`), en complément des
  variables d’environnement `COMKEY_*` actuelles — pour les réglages qu’on ne veut pas
  ré-exporter à chaque session.
- **`--json` étendu** à `list`/`tree`/`agents`, déjà présent sur `journal` — pour que d’autres
  outils consomment l’état sans parser du texte tabulaire.
- **`communikey provider test`/`provider list`** — détaillés plus haut, le vrai levier
  d’échelle sur les providers.

## Architecture de sécurité

Toute l’identité d’une session — Ed25519, X25519, ML-KEM-768 — dérive d’une **seule graine
maître de 32 octets**, par HKDF à domaines séparés (`crypto.go`). C’est l’unique secret : il
est scellé dans le vault, découpable en parts Shamir, ou encodable en phrase BIP-39. Toutes
les primitives viennent de la bibliothèque standard de Go 1.24 — zéro dépendance externe,
aucune crypto maison, à l’exception assumée des implémentations Shamir et BIP-39
(from-scratch, mais testées par roundtrip et par propriété de seuil).

La construction d’un message scellé, dans l’ordre : le secret de session est dérivé par
HKDF-SHA256 de l’échange X25519 combiné à l’encapsulation ML-KEM-768 vers la clé statique du
destinataire ; le texte est chiffré en AES-256-GCM avec un nonce aléatoire ; l’expéditeur
signe l’ensemble en **Ed25519 ⊕ ML-DSA-65** (les deux doivent être valides). À l’ouverture, les
DEUX signatures sont **vérifiées avant tout déchiffrement** — une enveloppe falsifiée, ou signée
avec un seul des deux schémas, est rejetée sans jamais atteindre la couche AEAD. Casser ce
secret exige de casser **les deux** échanges de clés, classique et post-quantique ; casser
l’authenticité exige de casser les **deux** schémas de signature : c’est la résistance visée
contre un adversaire qui capture aujourd’hui pour déchiffrer, ou qui usurpe demain, avec un
futur ordinateur quantique (« Harvest Now, Decrypt Later »). Le vault au repos est scellé en
AES-256-GCM, clé dérivée par **Argon2id** (RFC 9106, résistant GPU/ASIC), fichier en
permissions `0600`. Le réseau (`serve`/`remote`) chiffre le transport en TLS 1.3 hybride
post-quantique (`X25519MLKEM768`) avec certificat auto-signé Ed25519 et épinglage d’empreinte.

Le modèle de menace (`docs/THREAT-MODEL.md`) dit aussi, sans détour, ce que communikey **ne**
protège pas : la crypto n’a **pas encore reçu d’audit externe** — le vendre comme « le Signal
des agents » serait malhonnête, Signal est audité, communikey ne l’est pas encore. Sur une
même machine, sous le même utilisateur, la valeur du chiffrement de bout en bout est surtout
de la défense en profondeur : un attaquant qui exécute déjà du code sous cet identifiant lit
la mémoire du process de toute façon. L’agent qui **reçoit** un message reste une surface
d’injection de prompt à part entière — communikey transporte le message, il ne garantit pas
que l’agent destinataire ne sera pas détourné par son contenu ; cette défense relève du
harnais, pas du bus. Le journal expose les métadonnées (qui parle à qui, quand), jamais le
contenu — ce n’est pas de l’anonymat. La couche post-quantique couvre désormais l’échange de
clés (X25519 ⊕ ML-KEM-768) **et** les signatures (Ed25519 ⊕ ML-DSA-65, hybride, les deux
doivent être valides) — reste classique le certificat TLS auto-signé du transport
(`crypto/tls`/`x509` en Go n’acceptent pas encore de certificat feuille ML-DSA).

Le correctif le plus significatif de la branche non publiée mérite d’être cité précisément,
parce qu’il illustre bien le sérieux du projet sur ce terrain : un audit du 03/07 a trouvé que
`recovery combine`, sous le seuil requis, reconstituait un secret **bien formé mais faux** —
propriété documentée du schéma de Shamir — et comme n’importe quelle graine de 32 octets
dérive une identité qui a l’air valide, la commande écrasait le vault sans confirmation.
Un checksum SHA-256 tronqué à 4 octets, ajouté avant le découpage Shamir, rejette maintenant
toute reconstruction incorrecte avant qu’elle ne dérive quoi que ce soit ou n’écrase un
fichier existant.

## Face à la concurrence : hcom, pas ClaudeKit

Le vrai comparatif n’oppose pas communikey aux packs de capacités (ClaudeKit et consorts) —
ce sont des subagents *reviewers* intra-session, une autre catégorie qui ne fait pas ce que
communikey fait. Le concurrent réel est **hcom** (aannoo/hcom, MIT, environ 363 étoiles,
v0.7.22) : il parle déjà à dix CLIs, traverse les machines via un relais MQTT, et chiffre les
messages — mais avec une **clé pré-partagée**, sans authentifier l’expéditeur. hcom l’admet
lui-même : *« sender identity is routing metadata, not authorization »*. C’est exactement là
que communikey diffère structurellement — signature Ed25519 par expéditeur, chiffrement
asymétrique par destinataire, KEM post-quantique, recovery souveraine par Shamir et BIP-39.

| Capacité | communikey | hcom | Agent Teams (natif) |
|---|:---:|:---:|:---:|
| Cross-provider | ◐ (Claude fait, Codex/Gemini calibrés) | ✅ ~10 CLIs | ❌ Claude only |
| Cross-machine | ✅ TLS hybride PQC | ✅ MQTT | ❌ |
| Chiffrement | ✅ par destinataire | ✅ clé partagée | ❌ |
| Authentification de l’expéditeur | ✅ hybride Ed25519 ⊕ ML-DSA-65 | ❌ | ❌ |
| Post-quantique | ✅ ML-KEM-768 (échange) + ML-DSA-65 (signature) | ❌ | ❌ |
| Recovery souveraine | ✅ Shamir + BIP-39 | ❌ | ❌ |
| Largeur / maturité | alpha, un jour d’adoption | établi, plus large | natif, expérimental |

hcom gagne sur la largeur et la maturité ; communikey gagne sur la confiance. Les deux profils
sont à peu près des miroirs l’un de l’autre — et c’est un choix honnête plutôt qu’un problème à
cacher : le positionnement retenu (`docs/strategy/communikey-plan-lancement-haute-assurance.md`)
vise d’abord les équipes sécurité et les environnements régulés ou multi-tenant, ceux qui ne
peuvent justement pas se satisfaire d’un mot de passe partagé — pas une course de vitesse
contre hcom sur le nombre de CLIs supportés. Les releases signées et attestées (cosign
keyless, SLSA, SBOM) sont prévues à partir de 0.3.0, mais restent aujourd’hui bloquées par un
souci de facturation GitHub qui coupe aussi la CI et les Pages — un blocage réel, pas encore
levé.

## Ce qui n’est pas encore fait, honnêtement

| Phase | Livré | En route |
|---|---|---|
| Fondations | inbox coopératif, registre, journal, crypto E2E hybride PQC, vault AES-256-GCM | — |
| Terminaux et providers | injection cmux + tmux state-aware, graphe familial, détection Claude, adaptateurs Codex + Gemini (calibrés sur source, confirmation live en attente) | backend `screen`, passkey WebAuthn |
| Identité et réseau | recovery Shamir, phrase BIP-39, réseau loopback/LAN en TLS hybride PQC | authentification mutuelle réseau, durcissement hors-LAN |
| Portée | — | clients mobiles, Windows coopératif, pont Agent Teams, surface MCP |
| Durcissement | signatures hybrides Ed25519 ⊕ ML-DSA-65 | audit crypto externe, autres OS |

Trois blocages ne relèvent pas d’un manque de temps mais d’une contrainte réelle : le passkey
WebAuthn exige un authentificateur (navigateur, clé matérielle, API OS) qu’une session CLI
headless ne peut pas fournir pour être testé ; l’injection clavier sur Windows et sur mobile
est physiquement impossible, pas seulement non implémentée ; et calibrer un nouveau provider
exige de vrais écrans capturés — jamais des fixtures inventées, parce qu’un mauvais calibrage
soumettrait un message dans une session qui n’est pas prête à le recevoir.

## Ce à quoi tu n’as pas pensé

- ~~Boucle de calibration communautaire (`provider test`/`provider list`)~~ — **livré le 07-05**.
- **Mode démon** pour `serve`, en service `systemd` ou `launchd`, pour le réseau multi-machine,
  plutôt qu’un process lancé à la main à chaque fois. *(reste à faire)*
- **Export d’audit signé et horodaté** du journal — celui-ci existe déjà en hash seul ; une
  version signée prolongerait la posture « auditable » déjà revendiquée, sans jamais exposer
  le clair. *(reste à faire)*
- ~~Le coût double de chaque nouveau provider (détection d’état *et* snippet de hook,
  `hookInstallFor`)~~ — **corrigé le 07-05** : un provider inconnu affichait silencieusement le
  snippet Claude ; il affiche désormais un avertissement explicite et renvoie vers
  `communikey provider list`.

## Repères

- Modèle de sécurité complet : [`SECURITY.md`](../../SECURITY.md) · [`THREAT-MODEL.md`](../THREAT-MODEL.md)
- Comparatif détaillé : [`COMPARISON.md`](../COMPARISON.md) · [`communikey-axes-de-depassement.md`](communikey-axes-de-depassement.md)
- Plan de lancement : [`communikey-plan-lancement-haute-assurance.md`](communikey-plan-lancement-haute-assurance.md)
- Ce qui reste et pourquoi : [`docs/NEXT.md`](../NEXT.md)
- Câblage multi-éditeurs concret : [`docs/cross-vendor-setup.md`](../cross-vendor-setup.md)
- Version précédente de cette vision : [`communikey-vision-complete.md`](communikey-vision-complete.md) (2026-07-03)
- Journal des versions : [`CHANGELOG.md`](../../CHANGELOG.md)

---

**Auteur** : Aïssa BELKOUSSA · contact@aissabelkoussa.fr · Licence Apache-2.0
