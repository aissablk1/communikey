---
session_id: 6cb99775-csend-bus-universel
date_debut: 2026-06-27
date_fin: en cours
workspace: /Volumes/Professionnel/Projets/Développement/Outils/csend
auteur: Aïssa BELKOUSSA
statut: en cours
tags: [csend, inter-agent, architecture, crypto, pqc, memoire]
---

# Session 2026-06-27 — csend : du fix à la refonte « bus universel »

## QQOQCCP

- **Qui** : Aïssa BELKOUSSA (décisionnaire) ; exécution assistée.
- **Quoi** : analyser csend (outil d'injection inter-sessions Claude CLI), corriger ses
  faiblesses, le comparer au marché, puis le refondre en **bus de messagerie inter-agents
  universel** (cross-session, cross-provider, cross-terminal, cross-OS) avec mémoire persistante,
  crypto, vault passkey et recovery Shamir/BIP-39.
- **Où** : `…/Outils/csend` (binaire Go), config `~/.claude/skills/csend`, `~/.claude/commands/csend.md`.
- **Quand** : 2026-06-27.
- **Comment** : analyse de code + 2 agents de recherche (GitHub/Reddit) → brainstorming
  (skill superpowers) → spec → git → implémentation TDD par phases.
- **Combien** : chantier XL, livré par phases ; v1 = Phase 0 + Phase 1.
- **Pourquoi** : aucun outil du marché ne combine injection-dans-session-externe + state-aware
  + cross-provider + mémoire persistante. Trou de marché réel.

## Actions analysées

- csend actuel : Go, backend cmux (socket JSON-RPC), state-aware (idle/busy/confirm), graphe
  familial, audit PII-safe. Vérifié en live (`csend list` sur 9 sessions / 6 workspaces).
- Faiblesses confirmées : bug d'identité self (UUID env vs `surface:N` → garde anti-auto-injection
  contournable) ; couplage dur cmux ; détection fragile (3/6 sessions « unknown ») ; pas de git ;
  pas d'async/offline.
- Marché (vérifié) : injecteurs aveugles (Tmux-Orchestrator, MCP tmux) vs managers state-aware
  qui possèdent la session (ccmanager, CAO, agtx, tmai). Aucun ne couvre le croisement visé.
- Contraintes physiques : TIOCSTI mort, PTY slave ≠ stdin, seul le détenteur du PTY maître injecte.

## Actions réalisées

- 2026-06-27 — Analyse complète du code csend + log + skill + tests (lecture).
- 2026-06-27 — Vérif live : env cmux, `csend list`, `cmux tree --json` → bug self confirmé.
- 2026-06-27 — 2 agents de recherche (paysage GitHub ; Reddit + mécanismes techniques).
- 2026-06-27 — Brainstorming (skill) : architecture 8 couches validée par Aïssa.
- 2026-06-27 — `git init` du dépôt ; arborescence `docs/`.
- 2026-06-27 — Spec écrite : `docs/superpowers/specs/2026-06-27-csend-bus-universel-design.md`.
- 2026-06-27 — Journal de session créé (ce fichier).

## Actions à mener à l'avenir

- Phase 0 : fix bug self (TDD) ; Makefile ; CI ; PROJECT.nfo.
- Phase 1 : registre/mémoire persistante ; transport inbox coopératif ; crypto base ; router.
- Stretch : Shamir+BIP-39 ; backend tmux ; adaptateurs Codex/Gemini.

## Notes / Décisions / Blocages

- Décision : bus coopératif = backbone universel ; injection live = repli Unix-only.
- Décision : crypto = primitives auditées (stdlib Go 1.24 : ed25519/ecdh/mlkem/aes-gcm/hkdf/pbkdf2),
  jamais maison (§38). Recovery = BIP-39 + Shamir GF(256).
- Honnêteté (§29) : Windows/mobile = coopératif-only (pas d'injection). Livré par phases ;
  rapport final distingue testé vs conçu.
