---
title: "Fil « LanguageModel Protocol » → Ensemble : état de l'art vérifié + programme"
date: 2026-07-08
auteur: Aïssa BELKOUSSA
statut: contexte stratégique (pour tout agent futur, tout provider)
tags: [communikey, ensemble, lmp, strategie, landscape, recherche]
---

# LMP → Ensemble : le « pourquoi » stratégique (contexte pour agents futurs)

> But de ce fichier : qu'un agent de **n'importe quel provider** comprenne **pourquoi** le
> programme « Ensemble » existe, sur quelle recherche vérifiée il repose, et où est le pari.
> Complète le design SP0 (`docs/superpowers/specs/2026-07-08-ensemble-method-design.md`) et
> l'état courant (`docs/HANDOFF.md`). Faits vérifiés le 2026-07-08 ; le non-vérifié est marqué.

## 1. Généalogie du fil

Le fil « LanguageModel Protocol » part d'une recherche sur le protocole `LanguageModel` d'Apple
(WWDC, on-device) et son (absence de) rapport avec communikey → recadré en **client
multi-provider de modèles** (Phase 1, livrée) → puis, sur demande d'Aïssa, élargi en une
ambition : **une innovation catégorielle hybride protocole + méthode, à la BMAD**. Réponse
disciplinée : trouver le **wedge** réel (un angle tranchant), pas couvrir « tout » (§57).

## 2. État de l'art vérifié (mi-2026)

### Protocoles d'interopérabilité

| Protocole | Ce que c'est | Limite réelle |
|---|---|---|
| **MCP** (Anthropic → Linux Foundation) | couche **outil/contexte** pour *un* agent | standard de facto des outils, **pas** agent-à-agent ; surface d'injection documentée |
| **A2A** (Google → LF, 150+ orgs) | coordination **agent-à-agent** signée sur HTTP | conçu pour agents **web-service hébergés**, pas des sessions terminal ; adoption < MCP |
| **ACP** (IBM/BeeAI) | REST agent-à-agent | **fusionné dans A2A** (~août 2025), quasi-défunt en standalone |
| **AGNTCY / AGP** (Cisco → LF, 75+) | infra multi-agents « Internet of Agents » | pour **service-mesh cloud**, pas des agents de code en terminal |
| **Apple Foundation Models** | API Swift on-device | **silo fermé** Swift/Apple ; ouverture (`LanguageModel`, SDK Linux/Python) *annoncée été 2026, NON confirmée livrée* |
| **OpenAI Agents SDK** | successeur prod de Swarm ; a adopté MCP | **n'a PAS rejoint A2A** ; pas de coordination inter-agents ouverte |
| **Zed ACP** (Agent Client Protocol) | relie Claude Code/Codex/Gemini aux **éditeurs** | **éditeur↔agent**, PAS agent↔agent pair entre terminaux |

### Méthodes / frameworks agentiques

| Méthode | Douleur résolue → plainte |
|---|---|
| **BMAD** (~50 k★) | personas structurés → **coût token = plainte n°1** (~80-100 k tk/étape), faible legacy |
| **Spec-driven / Spec Kit** (~119 k★) | contexte durable → « **waterfall réinventé** » (test : 3h30 vs 23 min), friction synchro |
| **Subagents / superpowers** | contexte frais → **pas d'état partagé**, multiplicateur de tokens, latence |
| **claude-flow / ruflo** (~63 k★) | méta-harness → benchmarks **auto-déclarés jamais répliqués**, sur-ingénierie |
| **LangGraph / CrewAI / AutoGen** | orchestration in-process → non-déterminisme, abstractions qui fuient, **un « agent » = objet de config, pas un process terminal** |

**Constat clé** : aucune de ces méthodes ne coordonne de **vrais CLI de code hétérogènes entre
machines**. Elles orchestrent des appels LLM dans un seul process.

## 3. Les gaps (classés)

1. **Bus runtime *state-aware*, vendor-neutre, pour agents de code en terminal** — LE gap réel.
   L'« agentmaxxing » (Claude Code + Codex + Gemini en parallèle) repose sur worktrees + humain
   standardiste ; les agents ne peuvent pas se parler sans corrompre un prompt occupé, sans
   identité crypto. **L'intersection control-socket + busy/idle/confirm + cross-vendor + identité
   signée est ABSENTE des repos publics.** (Les mailboxes GitHub : stateless/non-signées ou
   mono-vendeur.) → **c'est le créneau de communikey.**
2. **Fiabilité / vérification** — réel mais **BONDÉ** (tout le monde l'attaque). Ne pas y aller frontalement.
3. **Identité/authz runtime d'un mesh local** — réel, **feature du Gap 1**, pas produit isolé.
4. **Efficience token / brownfield** — méthodologique, différenciation faible.

## 4. Verdict / wedge

**Un seul pari catégoriel : le Gap 1** — « le TCP des agents de terminal » : protocole +
implémentation de référence pour la coordination *state-aware*, cross-vendor, d'agents de code.
communikey occupe **déjà** ce créneau ; il faut le transformer en **méthode adoptable + standard
émergent**. Le moat de la vérification : **writer≠checker CROSS-VENDOR** (un modèle d'un autre
vendeur ne partage pas les points aveugles du builder — théorie de l'*ensemble*). À **prouver en
dogfood**, pas à affirmer.

Ne PAS construire : un énième orchestrateur UX (claude-squad, vibe-kanban *sunsetting*, omnara
*archivé*, crystal *déprécié* — segment qui meurt déjà).

## 5. Le programme (séquencé — « tout réaliser » dans l'ordre)

- **SP0 — Méthode « Ensemble »** *(design fait)* : la recette nommée (rôles Architecte/Builders/
  Checker/Chef d'orchestre) sur le bus. v1 = méthode + primitives minimales + **dogfood**.
- **SP1 — Implémentation de référence** : primitives sur communikey (rôles, handoff de slice).
- **SP2 — Protocole ouvert** : format de fil publié, **adoption-led** (gagner comme MCP/BMAD, pas décréter).
- **SP3 — Control-plane commercial** : SaaS équipe (policy/audit/RBAC/revue).

## 6. Risques honnêtes (§34)

- **Course au standard** : lente, dépendante de l'adoption des vendeurs.
- **Absorption** : Anthropic/OpenAI/Google ou une extension « vers le bas » d'A2A peuvent avaler
  le créneau. Fenêtre = maintenant.
- **Niche aujourd'hui** : power-users. Pari sur une vague qui se normalise.
- **Le moat writer≠checker cross-vendor est une HYPOTHÈSE** tant que le dogfood ne l'a pas prouvé.

## 7. Sources & honnêteté (§29)

Recherche menée le 2026-07-08 (agent dédié, ~14 fetches). **Vérifié sur source primaire** : A2A→LF
(developers.googleblog.com, siliconangle), ACP→A2A (lfaidata.foundation), AGNTCY (linuxfoundation),
star counts GitHub live (LangGraph/CrewAI/AutoGen/BMAD/Spec Kit/ruflo…), Zed ACP, OpenAI adopte MCP
(techcrunch), Apple Foundation Models (machinelearning.apple.com).
**NON vérifié / incertain** : ouverture Apple `LanguageModel` (annoncée, non confirmée) ; « MCP 97 M
dl/mois » (auto-déclaré) ; benchmarks claude-flow (auto-déclarés, non répliqués) ; « adoption A2A
78 % vs 23 % » et « < 15 % des pilotes en prod » (sources secondaires) ; funding CrewAI 18 M$ (secondaire).
Un agent futur doit **re-vérifier** avant de citer comme fait (le paysage bouge vite).

---

**Auteur** : Aïssa BELKOUSSA · Apache-2.0
