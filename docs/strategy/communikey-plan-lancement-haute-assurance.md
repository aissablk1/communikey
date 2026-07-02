---
title: "communikey — Plan de lancement « haute assurance »"
date: 2026-07-01
auteur: Aïssa BELKOUSSA
projet: communikey
version: 1.0
statut: validé
tags: [stratégie, lancement, gtm, positionnement, haute-assurance]
---

# communikey — Plan de lancement « haute assurance »

> GTM (skill `launch-strategy`) pour la **seule niche où communikey est structurellement supérieur** :
> le bus d'agents **authentifié + post-quantique + souverain**. Ancré sur la réalité (§29/§2) :
> communikey est un **alpha solo, ~0 adoption, gratuit**, face à `hcom` (plus mûr, plus large). Pas de
> faux chiffre, pas de faux-semblant de traction.

## 0. La réalité de départ (honnête)

- **Produit** : alpha `v0.2.0`, **v0.3.0 crypto-breaking à venir** (transcript signé — surface:45).
- **Audience owned** : quasi nulle (repo d'un jour, aucune liste e-mail, aucun trafic).
- **Bloqueurs actifs** : facturation GitHub → **CI rouge + Pages OFF** ; **communikey.dev pas en ligne**.
- **Concurrent** : `hcom` (10 CLIs, cross-machine, chiffré PSK, ~363★) — **plus mûr et plus large**.

**Conclusion de cadrage :** ne PAS lancer maintenant (badge rouge, pas de site, crypto qui va casser).
Le lancement se **mérite** — d'abord la crédibilité technique, pas le bruit.

## 1. Positionnement (énoncé)

> Pour les **opérateurs multi-agents en contexte sensible** qui **ne peuvent pas faire confiance à
> un bus « à mot de passe partagé »**, **communikey** est le bus inter-agents à **vraie identité
> cryptographique** : il **authentifie chaque expéditeur** (Ed25519 signé), **chiffre pour le
> destinataire** (X25519 ⊕ ML-KEM-768, post-quantique) et offre une **recovery souveraine** —
> contrairement à `hcom` et aux autres bus (PSK partagé ou clair, sans auth d'expéditeur).

## 2. ICP (par ordre de priorité)

1. **Sécurité / red-team / netsec** qui orchestrent plusieurs agents et veulent une **provenance
   auditable** (« qui a envoyé cet ordre ? »). C'est le cœur : ils *comprennent* pourquoi un PSK
   partagé ne suffit pas.
2. **Environnements régulés / multi-tenant / agences** (finance, santé-adjacent, secteur public,
   prestataires) : besoin d'isolation, de souveraineté (self-host, UE §58), de non-répudiation.
3. **Souverainistes / anti-lock-in** (proches de tes règles §58) : libre, auto-hébergeable, PQC.
4. (Élargissement plus tard) power-users multi-agents génériques — mais là hcom gagne sur la
   simplicité ; ne pas y aller frontalement au début.

## 3. Messaging (par audience)

- **netsec** : *« Ton bus d'agents sait-il QUI parle ? Un PSK partagé, non. communikey signe chaque
  message — provenance, pas juste confidentialité. »*
- **régulé/souverain** : *« Coordination d'agents auto-hébergée, chiffrée par destinataire,
  post-quantique, zéro dépendance, zéro lock-in. »*
- **honnêteté obligatoire (§34/§29)** : reconnaître la maturité de hcom, ne revendiquer QUE le vrai
  avantage (auth + PQC + recovery + souveraineté). Zéro chiffre d'adoption inventé, zéro témoignage.

## 4. Carte de canaux ORB

| Type | Canal | Tactique |
|---|---|---|
| **Owned** | Repo GitHub | README (wedge ✅), SECURITY.md (modèle de menace ✅), page **COMPARISON** honnête vs hcom |
| **Owned** | Site `communikey.dev` | Hero recadré (✅) ; **à héberger** (Cloudflare Pages, indépendant de la facturation) |
| **Owned** | Post technique | Write-up « pourquoi un PSK partagé ne suffit pas pour des agents » (le moat, pédagogique) |
| **Rented** | **r/netsec, r/crypto, r/LocalLLaMA** | l'angle sécurité/souveraineté (PAS r/ClaudeAI en premier — mauvais public pour le moat) |
| **Rented** | **Show HN** | *« communikey — the authenticated, post-quantum message bus for AI agents »* — **après** v0.3.0 + site + CI verte |
| **Rented** | X (dev/sécurité) | thread « signé vs mot de passe partagé » avec la démo réelle 2 agents |
| **Borrowed** | Discussions bus d'agents | S'inserer là où le public est (issues/threads hcom, murmur, Agent Teams) — apport, pas spam |
| **Borrowed** | awesome-lists | PR **plus tard** (quand v0.3.0 mûr) — un alpha se fait rejeter |

## 5. Phases (séquencées, gatées)

| Phase | Pré-requis | Actions | Métrique |
|---|---|---|---|
| **0 — Débloquer** | — | Facturation GitHub (CI+Pages) ; v0.3.0 (surface:45) ; site sur Cloudflare Pages | CI verte, site en ligne |
| **1 — Crédibilité** | Phase 0 | Post technique « modèle de sécurité / provenance » ; page COMPARISON honnête vs hcom ; poster **r/netsec** + X | 3-10 **vrais** testeurs de la niche + retours |
| **2 — Show HN** | v0.3.0 poli + site + quelques testeurs réels | Show HN (angle authentifié/PQC) ; répondre honnêtement (dont « vs hcom ») | commentaires qualifiés, 1ers vrais usages |
| **3 — Momentum** | après HN | Approfondir la niche (régulé/souverain), retours → roadmap, awesome-lists si mûr | rétention réelle, contributeurs |

## 6. Ce qu'il NE faut PAS faire (§34/§57)

- **Pas de Product Hunt en alpha** (CI rouge, pas de site, v0.3.0 qui casse) → ça grille la 1ʳᵉ impression.
- **Pas de fausse traction** (étoiles/témoignages/chiffres inventés) — 1 vrai testeur netsec > 100 faux.
- **Pas de guerre de largeur vs hcom** au début — on perd. On gagne sur la **confiance**.
- **Pas de spam awesome-lists** d'un projet d'un jour.

## 7. Prochaine action concrète (dépendances)

Le lancement est **gaté par 2 choses hors de mon territoire** : la **facturation GitHub** (toi) et la
**v0.3.0** (surface:45). En attendant, le seul livrable owned qui manque et que je peux faire est la
**page COMPARISON honnête vs hcom** (battle-card publique, distincte de l'analyse interne). Le reste
(héberger le site, poster, PR) dépend de tes comptes / de la maturité v0.3.0.
