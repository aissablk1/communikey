---
title: "Ensemble — méthode de flottes d'agents de code cross-vendor, state-aware et souveraines"
date: 2026-07-08
auteur: Aïssa BELKOUSSA
statut: brouillon (design SP0 — en attente de revue)
tags: [communikey, ensemble, methode, multi-agent, cross-vendor, design]
---

# Ensemble — SP0 : la Méthode

> **Nom de travail : « Ensemble »** (révisable). Double sens assumé : l'*ensemble*
> statistique (combiner des modèles **divers** bat le meilleur modèle seul) et
> l'*ensemble* musical (un chef coordonne des instruments **hétérogènes**). Les deux
> décrivent la thèse : la diversité de vendeurs, coordonnée, produit mieux qu'un seul.

## Résumé

Ensemble est une **méthode nommée** (à la BMAD / subagent-driven) pour piloter une
**flotte hétérogène et persistante** d'agents de code — chaque rôle sur le vendeur le
plus fort/le moins cher pour lui (Claude, Codex, Gemini, Antigravity, ClawCodex…) —
**coordonnée par un bus signé, chiffré, state-aware** (communikey, qui en est
l'implémentation de référence). L'humain devient **chef d'orchestre, pas standardiste.**
C'est SP0 du programme « LanguageModel Protocol » : la couche *adoption* qui rend le
reste (implémentation de référence, protocole ouvert, control-plane commercial)
atteignable — l'ordre par lequel BMAD et MCP ont réellement gagné.

## 1. Le problème (tranchant, vérifié)

Le pattern « agentmaxxing » de 2026 — des devs seniors lançant Claude Code + Codex +
Gemini en parallèle — repose aujourd'hui sur **git worktrees + un humain-standardiste**.
Concrètement, quatre douleurs :

1. **Les agents ne peuvent pas se parler sans se corrompre** : injecter un message dans
   une session occupée ou sur une boîte de confirmation casse l'état de la cible.
2. **Aucune identité** : rien ne dit *qui* parle — juste « quelqu'un avec le bon socket ».
3. **Coût token** : re-spawner des subagents frais recharge le contexte à chaque fois
   (plainte n°1 de BMAD : ~80–100 k tokens/étape ; subagents = multiplicateur de tokens).
4. **L'humain est le câble** : coordination = copier-coller manuel entre fenêtres.

**Ce que la recherche confirme (2026-07-08)** : la consolidation s'est faite autour de
**MCP** (couche outils) et **A2A** (agent-à-agent, Linux Foundation), mais A2A vise des
agents **web-service hébergés** (HTTP/gRPC), Zed ACP relie **éditeur↔agent**, et les
mailboxes GitHub (mcp_agent_mail, agmsg…) sont **stateless/non signées ou mono-vendeur**.
**L'intersection exacte — control socket + conscience busy/idle/awaiting-confirmation +
cross-vendor + identité cryptographique — est absente des repos publics.** C'est le
créneau que communikey occupe déjà, et qu'Ensemble transforme en méthode.

## 2. La thèse

> Une **flotte hétérogène et persistante** de CLI de code, chaque rôle chez le vendeur
> qui l'exécute le mieux/le moins cher, **coordonnée par un bus state-aware** qui n'injecte
> jamais dans un prompt occupé, signe qui parle, et traite le repo comme mémoire partagée.

Deux mots portent la différence : **hétérogène** (vendeurs différents, pas des personas
d'un même modèle) et **persistant** (sessions déjà vivantes, pas des subagents éphémères).

## 3. Ce qui est nouveau

| Face à | Leur modèle | Faiblesse (vérifiée) | Ce qu'Ensemble change |
|---|---|---|---|
| **BMAD** | personas mono-vendeur, éphémères | coût token = plainte n°1 ; faible sur le legacy | vendeurs réels routés par force/coût ; sessions persistantes |
| **Subagent-driven** | subagents à contexte frais | **pas d'état partagé** ; multiplicateur de tokens ; boucles retirées à ~25 min | sessions vivantes coordonnées par bus, pas rechargées |
| **A2A / AGNTCY** | agents web-service (HTTP/gRPC) | pas de sessions terminal interactives | CLI de code réels, en terminal, state-aware |
| **CrewAI / AutoGen** | « crews » in-process, single-process | ni monitoring, ni récupération, ni identité | processus terminaux distincts, signés, cross-machine |

## 4. Les rôles & la boucle

Formalisation directe des règles §40 (Architect/Builder) et §41 (writer≠checker) d'Aïssa :

- **Architecte** — vendeur cher (ex. Claude/Opus). Juge l'état, arbitre, écrit la spec de
  la prochaine *slice*, **n'écrit jamais de code d'implémentation**.
- **Builders** — vendeurs rapides/bon marché (ex. Codex), en parallèle sur **modules
  isolés** (≤ 3-4 lanes). Exécutent la slice.
- **Checker** — un vendeur **différent** du builder (ex. Gemini). Vérif **adversariale**
  (instruit pour *réfuter*). Ne note jamais sa propre copie.
- **Chef d'orchestre** — l'humain. Décide kill/continue, fournit les inputs que lui seul a.

**La boucle** : Architecte lit `HANDOFF.md` → écrit la slice → `communikey send` (gaté :
n'injecte que si le builder est *idle*) → Builders exécutent, commit, mettent à jour
`HANDOFF.md` en **résultats bruts** → Checker (autre vendeur) réfute → Architecte juge
les résultats bruts contre des *gates* gelés → slice suivante. Le bus route par état, le
repo est la mémoire, les handoffs sont signés.

## 5. Le moat : writer ≠ checker CROSS-VENDOR

Le différenciateur défendable. La théorie de l'*ensemble* (ML) : des modèles aux **erreurs
décorrélées** combinés surpassent le meilleur modèle seul. Appliqué à la vérification :
**un bug que le modèle-builder se justifie à lui-même, un modèle d'un autre vendeur ne
partage pas ce point aveugle.** metaswarm bricole ça à la main ; personne ne l'a industrialisé
via un bus. C'est une **hypothèse à valider empiriquement** (cf. §8 dogfood), pas un acquis.

## 6. Ancrage sur communikey (primitives existantes)

Ensemble n'invente pas de transport — il **compose** ce que communikey livre déjà :

- `communikey send <cible> "…"` — livraison **gatée par l'état** (idle/busy/confirm).
- `communikey read <cible>` / `list` / `tree` — lire l'état & le graphe familial.
- `communikey send --down` — broadcast aux builders enfants.
- Identité **Ed25519** signée + chiffrement E2E PQC — *qui* parle, auditable.
- `HANDOFF.md` versionné — mémoire partagée (résultats bruts, zéro interprétation).

**Primitives minimales à ajouter en v1** (le strict nécessaire, §57) : une convention de
**rôle** (architecte/builder/checker) portée dans le registre de session, et une commande
de **handoff de slice** qui écrit `HANDOFF.md` + notifie le bon rôle. Rien de plus.

## 7. Périmètre v1 (décidé — autonome, §66/§57)

**Dans la v1 :**
- La **méthode documentée** : thèse, rôles, boucle, règles invariantes (guide teachable).
- Les **primitives minimales** ci-dessus sur communikey.
- **Validation par dogfood** : une vraie tâche menée par une flotte hétérogène
  (Claude+Codex+Gemini via communikey) — la méthode s'écrit *depuis le vécu* (§2).

**Explicitement reporté (SP1+), pas jeté :**
- Routage automatique par force/coût, tableau de bord de flotte, control-plane managé.
- Le protocole ouvert publié (SP2) — émerge de l'usage, pas décrété.

## 8. Critères de succès

1. Une **vraie tâche non triviale** est livrée par une flotte hétérogène coordonnée par le
   bus, **sans humain-standardiste** (l'humain ne fait que kill/continue/inputs).
2. Le **checker cross-vendor attrape ≥ 1 défaut réel** que le builder avait validé — preuve
   du moat (§5), montrée sur sortie réelle (§29), pas affirmée.
3. Coût token **mesuré** vs. un run mono-vendeur équivalent (l'argument « persistant < éphémère »
   doit être **chiffré**, pas supposé — §2/§34).
4. La méthode est **répétable** : un tiers peut la suivre depuis le guide seul.

## 9. Risques & limites honnêtes (§34)

- **Course au standard** : lente, dépendante de l'adoption des vendeurs. On gagne par
  l'implémentation, pas par décret.
- **Absorption** : Anthropic/OpenAI/Google, ou une extension « vers le bas » d'A2A, peuvent
  avaler le créneau. Fenêtre = maintenant.
- **Niche aujourd'hui** : power-users « agentmaxxing ». Pari sur une vague.
- **Contraintes techniques héritées de communikey** : injection live **Unix-only** (Windows
  natif ❌) ; dépend de CLI de vendeurs restant scriptables ; crypto **non auditée** (alpha).
- **Le moat est une hypothèse** tant que le critère 2 n'est pas prouvé en dogfood.

## 10. Place dans le programme

`SP0 Méthode (ici)` → `SP1 implémentation de référence (primitives + outillage)` →
`SP2 protocole ouvert publié (adoption-led)` → `SP3 control-plane commercial`. Chaque
sous-projet a sa propre spec → plan → build. « Tout réaliser » = cette séquence, pas la
simultanéité.

## 11. Questions ouvertes

1. **Nom définitif** : « Ensemble » vs « Orchestra/Conductor » vs « Fleet » vs autre.
2. **Marque/repo** : Ensemble reste-t-il dans communikey, ou graduera-t-il en repo/marque
   propre (comme BMAD est séparé de tout outil) ?
3. **Vendeurs cibles du dogfood** : Claude+Codex+Gemini (adaptateurs déjà livrés) — confirmer.
4. **Assignation de rôle par défaut** : qui est architecte/builder/checker dans TON setup réel ?

---

**Auteur** : Aïssa BELKOUSSA
