---
title: "Ensemble — la méthode (SP0, opérationnelle)"
date: 2026-07-09
auteur: Aïssa BELKOUSSA
statut: validé (décisions arrêtées par délégation d'Aïssa « fais tout pour moi », 2026-07-09)
tags: [ensemble, methode, communikey, multi-agent, cross-vendor, dogfood]
---

# Ensemble — la méthode

> Piloter une **flotte hétérogène et persistante** d'agents de code — chaque rôle chez le vendeur le
> plus fort/le moins cher — coordonnée par le **bus state-aware communikey**. L'humain est **chef
> d'orchestre, pas standardiste.** Design complet : `docs/superpowers/specs/2026-07-08-ensemble-method-design.md`.

## 0. Décisions arrêtées (2026-07-09)

- **Nom** : **Ensemble** (double sens : *ensemble* ML — des modèles divers battent le meilleur seul ;
  *ensemble* musical — un chef coordonne des instruments hétérogènes).
- **Runtime** : la méthode tourne sur les **primitives communikey EXISTANTES** (aucun nouveau code).
- **Rôles** : Architecte / Builders / Checker (vendeur ≠ builder) / Chef d'orchestre (humain).
- **Moat** : **writer ≠ checker CROSS-VENDOR** — à prouver en dogfood (§4), pas à affirmer.

## 1. Les rôles → mapping vendeur

Le vendeur est un *paramètre*, pas un dogme. Mapper selon ce qui tourne réellement chez toi :

| Rôle | Fonction | Vendeur suggéré (à adapter) |
|---|---|---|
| **Architecte** | Juge l'état, écrit la spec de *slice*, arbitre. **N'écrit jamais de code.** | Claude/Opus (jugement) |
| **Builders** (≤ 3-4) | Exécutent la slice sur **modules isolés**, en parallèle. | Codex, ou tout CLI rapide/bon marché |
| **Checker** | Vérif **adversariale** (instruit pour *réfuter*). Vendeur **DIFFÉRENT** du builder. | Gemini / Antigravity (autre vendeur) |
| **Chef d'orchestre** | Décide kill/continue, fournit les inputs que lui seul a. | Toi (humain) |

> Hermes Agent (Nous Research) est désormais **supporté** par communikey (adaptateur `provisoire`) —
> utilisable comme **builder/checker supplémentaire** ou **rôle mémoire** (sa *learning loop*), une
> fois son `provisoire` levé par une capture live.

## 2. La boucle (runnable sur communikey existant)

Chaque session d'agent = un panneau/terminal, rejoint le bus (`communikey register`). Le repo est la
mémoire (`docs/HANDOFF.md`, §40). Toutes les commandes existent déjà.

```
1. Architecte lit HANDOFF.md → écrit la slice (spec + gates gelés) dans HANDOFF.md.
2. Architecte notifie les builders :  communikey send <builder> "slice prête: <réf HANDOFF>"
   (livraison GATÉE : n'injecte QUE si le builder est idle — jamais dans un prompt occupé/confirm).
3. Builders exécutent leur module, commit, écrivent des RÉSULTATS BRUTS dans HANDOFF.md.
4. Architecte notifie le Checker :  communikey send <checker> "réfute la slice <réf> — cherche un bug réel"
   → le Checker (vendeur DIFFÉRENT) vérifie en adversaire.
5. Architecte lit l'état de la flotte :  communikey list / tree / read <cible>
   juge les résultats bruts contre les gates, décide slice suivante ou kill.
6. Répéter. Le bus route par état, le repo est la mémoire, les handoffs sont signés (identité Ed25519).
```

Invariants (hérités du bus) : jamais d'injection dans un prompt occupé ou une confirmation ; idle
requis pour auto-valider ; chaque message signé (on sait *qui* parle).

## 3. Règles invariantes (de §40/§41)

1. Le **repo est la mémoire** — pas dans HANDOFF.md = ça n'a pas existé.
2. Le builder **ne note jamais** sa propre copie (séparation juge/exécutant).
3. Le **désaccord est obligatoire** (conformité silencieuse = échec).
4. **Geler les critères de succès AVANT** les résultats — jamais les éditer après.
5. **Checker ≠ vendeur du builder** (le cœur — §4).

## 4. Le moat : writer ≠ checker CROSS-VENDOR

Théorie de l'*ensemble* : des modèles aux **erreurs décorrélées** combinés surpassent le meilleur
seul. Appliqué à la vérification : **un bug que le modèle-builder se justifie à lui-même, un modèle
d'un AUTRE vendeur ne partage pas ce point aveugle.** C'est une **hypothèse à prouver** (dogfood §5),
pas un acquis (§34).

## 5. Dogfood #1 — turnkey (à LANCER par le chef d'orchestre)

Objectif : livrer une vraie petite tâche par une flotte hétérogène **sans humain-standardiste**, et
**mesurer** si le checker cross-vendor attrape un défaut réel + le coût token vs mono-vendeur.

**Prérequis (côté Aïssa — je ne peux pas les lancer headless, §2)** :
- 3 terminaux/panneaux avec 3 vendeurs DIFFÉRENTS configurés (clés/OAuth) et lancés, rejoints au bus
  (`communikey register` dans chacun). Ex. : Architecte=Claude, Builder=Codex, Checker=Gemini.
- Un dépôt Git de travail (worktrees pour isoler les builders), `docs/HANDOFF.md` scaffoldé.

**Tâche exemple (remplaçable)** : une feature petite, bien bornée, à critères durs — ex. « ajouter
une sous-commande CLI `X` avec 2 tests ». Assez petite pour 1 slice, assez réelle pour qu'un bug soit
possible.

**Script** :
1. Chef : `communikey register` dans chaque panneau ; `communikey list` (voir les 3 idle).
2. Architecte : écrit la slice + gates dans `HANDOFF.md` ; `communikey send <builder> "slice prête"`.
3. Builder : implémente + tests, commit, écrit résultats bruts dans `HANDOFF.md`.
4. Architecte : `communikey send <checker> "réfute la slice — bug réel ou cas limite non couvert ?"`.
5. Checker (autre vendeur) : cherche un défaut, écrit son verdict brut dans `HANDOFF.md`.
6. Architecte : juge contre les gates ; slice suivante ou stop.

**Critères de succès (à mesurer, §29 — montrer la sortie réelle)** :
- [ ] Tâche livrée par la flotte, coordonnée **par le bus** (le chef ne fait que kill/continue/inputs).
- [ ] Le **checker cross-vendor attrape ≥ 1 défaut réel** que le builder avait validé → **preuve du moat**.
- [ ] Coût token **chiffré** vs un run mono-vendeur équivalent (l'argument « persistant < éphémère »).
- [ ] La méthode est **répétable** depuis ce guide seul.

## 6. Ce qui nécessite Aïssa (frontière honnête)

Je ne peux **pas** exécuter le dogfood à ta place : il faut de **vrais terminaux multi-vendeurs
vivants** sur ta machine (je ne spawne pas de sessions Codex/Gemini/Hermes headless), et je **refuse
de simuler** un run avec des sorties inventées (§2 zéro-mock). Tout le reste est **prêt**. Quand tes
3 panneaux sont lancés et enregistrés au bus, dis-le moi : je te guide pas-à-pas et je joue
l'**Architecte** en live.

---

**Auteur** : Aïssa BELKOUSSA · Apache-2.0
