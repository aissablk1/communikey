# csend v2 — Bus de messagerie inter-agents universel — Design

- **Projet** : csend
- **Auteur** : Aïssa BELKOUSSA · contact@aissabelkoussa.fr
- **Date** : 2026-06-27
- **Statut** : design validé — implémentation par phases
- **Tags** : inter-agent, cli, claude-code, transport, crypto, pqc, mémoire

---

## 1. Problème & trou de marché (vérifié)

Il n'existe **aucune API officielle** pour injecter un prompt dans une session d'agent CLI
(Claude Code, Codex, Gemini…) **en cours**. Confirmé par les feature requests officielles
fermées/ouvertes : `claude inject` ([#24947](https://github.com/anthropics/claude-code/issues/24947),
fermée), [#27441](https://github.com/anthropics/claude-code/issues/27441),
[#21419](https://github.com/anthropics/claude-code/issues/21419),
[#37213](https://github.com/anthropics/claude-code/issues/37213).

Contraintes physiques (vérifiées) :
- `TIOCSTI` (simuler des frappes) est **mort** : désactivé par défaut sous Linux ≥ 6.2
  (`dev.tty.legacy_tiocsti=0`), **root-only (EPERM)** sur macOS, supprimé d'OpenBSD.
- Écrire dans le **PTY slave** (`/dev/pts/N`) part vers l'**affichage**, jamais dans le stdin
  du process.
- **Seule voie générique** d'injection : un programme qui **détient le PTY maître** de la cible
  (tmux/screen/cmux) pousse les octets — ce que fait `tmux send-keys`.

**Paysage (vérifié)** : les vrais injecteurs (Tmux-Orchestrator, MCP tmux) envoient **à
l'aveugle** ; les outils *state-aware* (ccmanager, awslabs CAO, agtx, tmai) sont des **managers
qui possèdent la session** parce qu'ils l'ont spawnée. **Le croisement « injecter dans une
session externe vivante ET lire son état d'abord ET cross-provider ET mémoire persistante »
n'existe nulle part.** C'est la place de csend.

## 2. Idée-pivot (ce qui rend « tous » physiquement vrai)

> **Le bus coopératif (inbox) est la colonne vertébrale universelle.
> L'injection clavier live est un backend de dernier recours, Unix-only.**

- Une session qui **coopère** (a le hook/MCP/listener csend) est joignable **partout** : tout OS,
  tout provider, Agent Teams, sous-agents, distant, mobile.
- Une session **muette** (lancée à la main, sans csend) reste joignable par **injection live**
  tant qu'elle tourne dans un multiplexeur Unix (cmux/tmux/screen).

Une seule adresse, un seul outil, et le message trouve toujours un chemin.

## 3. Architecture — 8 couches

Chaque couche = une unité à responsabilité unique, interface explicite, dépendances bornées.

```
┌── COUCHE 0 — SÉCURITÉ & CRYPTO (transverse) ──────────────────────┐
│ identité = clés (Ed25519 sign + X25519 KEM, + ML-KEM/ML-DSA PQC)   │
│ messages E2E chiffrés+signés (le bus relaie du chiffré, zero-trust)│
│ vault local (AES-256-GCM, clé via PBKDF2/HKDF→Argon2id) + passkey  │
│ recovery : BIP-39 + Shamir N-sur-M (GF(256)). Primitives auditées. │
└────────────────────────────────────────────────────────────────────┘
  7. SURFACES   CLI `csend` · Skill (par session) · Hook receive · MCP
  6. MÉMOIRE    store durable interrogeable (messages+registre+états) ⇄ claude-mem
  5. RÉSEAU     daemon/machine ; fédération auth+TLS hybride PQC (mono = socket local)
  4. ROUTEUR    choisit le chemin de livraison (matrice §4)
  3. TRANSPORTS a) inbox coopératif (universel)  b) injection live (cmux/tmux/screen, Unix)
                c) bridges natifs (Agent Teams mailbox)
  2. PROVIDERS  adaptateurs d'état idle/busy/confirm (Claude✓ · Codex · Gemini · générique)
  1. IDENTITÉ   adresse stable agent://<machine>/<provider>/<session> + registre + graphe familial
```

### Frontières des unités (résumé)
- **identity/** : résolution d'une adresse stable ; corrige le bug self (source faisant autorité).
- **registry/** : sessions connues + relations père/enfants (persisté).
- **transport/** : contrat adressé-par-surface ; impls `cmux`, `tmux`, `inbox`.
- **provider/** : `DetectState(screen) → idle|busy|confirm|unknown`, un par provider.
- **router/** : `Route(target, msg) → chemin` selon la matrice §4.
- **crypto/** : clés, seal/open, vault, recovery.
- **memory/** : journal append-only + requêtes ; pont claude-mem.
- **net/** : daemon + fédération (phase ultérieure).

## 4. Matrice de routage (couche 4)

| Cible | Chemin | État via |
|---|---|---|
| Coopère (hook/MCP/listener) | **inbox** (durable, tout OS/provider) | hooks = état autoritaire |
| TUI Unix muette dans un multiplexeur | **injection live** (cmux/tmux/screen) | scraping d'écran |
| Agent Teams (in-process) | **mailbox drop** (`~/.claude/teams/…`) | statut équipe |
| Autre machine | **réseau** → daemon distant → livraison locale | relayé |
| Offline | **file** (mémoire/inbox), livrée à la reconnexion | — |

## 5. Sécurité & cryptographie (couche 0)

**Garde-fou n°1 (§38) : primitives AUDITÉES, jamais maison.**

| Besoin | Primitive | Source (Go 1.24) |
|---|---|---|
| Signature | Ed25519 (+ ML-DSA PQC à venir) | `crypto/ed25519` (stdlib) |
| Échange de clés | X25519 + **ML-KEM-768 hybride** | `crypto/ecdh` + `crypto/mlkem` (stdlib) |
| Chiffrement | AES-256-GCM | `crypto/aes`+`crypto/cipher` (stdlib) |
| KDF | HKDF / PBKDF2 (→ Argon2id en option) | `crypto/hkdf`,`crypto/pbkdf2` (stdlib) |
| Aléa | CSPRNG | `crypto/rand` (stdlib) |
| MFA vault | **Passkey WebAuthn/FIDO2** (PRF) | client (phase mobile/desktop) |
| Recovery | **BIP-39** mnémonique + **Shamir** N-sur-M | impl. testée GF(256) |

- **Messages** : chiffrés de bout en bout + signés. Le bus/relais ne voit que du chiffré.
- **Vault** : coffre local des clés/secrets, chiffré au repos, déverrouillé par passkey
  (résistant au phishing, §38.7 — jamais d'OTP SMS).
- **Recovery Shamir** : seed → N parts, seuil M-sur-N (ex. 3/5) ; aucune part isolée ne révèle
  rien (sûreté de seuil information-théorique). Implémentation from-scratch **testée**
  (roundtrip + propriété de seuil) ; audit externe recommandé avant usage critique (§29/§38).
- **Crypto-agilité** : l'algo est remplaçable sans réécrire le protocole.

## 6. Périmètre cross-OS — la vérité (§2/§29)

| Plateforme | Bus coopératif | Injection live |
|---|---|---|
| macOS / Linux / WSL / Chromebook-Crostini | ✅ | ✅ (cmux/tmux/screen/PTY) |
| Windows natif | ✅ | ❌ (pas de PTY maître partageable ; ConPTY ne s'attache pas) |
| iOS / iPadOS | ✅ (client du bus) | ❌ (sandbox) |
| Android | ✅ (client) | ❌ live cross-app (Accessibility = hors-scope) |
| Autres (BSD, conteneurs, CI) | ✅ s'ils parlent au bus | selon multiplexeur présent |

Sur mobile : voir l'état de chaque session, recevoir « session X attend une confirmation »,
**répondre/approuver depuis le téléphone**, broadcaster à la famille — en E2E chiffré.

## 7. Bug de sûreté corrigé en Phase 0

`selfRef()` préfère `CMUX_SURFACE_ID` (un **UUID**) alors que les cibles sont en forme
`surface:N` ; le garde `tgt.Ref == self` ne matche jamais → l'anti-auto-injection est
contournable. **Fix** : source d'identité faisant autorité = `cmux tree` `caller.surface_ref`
(même espace `surface:N`) et/ou le flag `here=True`. Test de non-régression obligatoire.

## 8. Non-goals / limites honnêtes

- Pas d'injection clavier sur Windows natif, iOS, iPadOS, Android (impossible/sandbox) →
  ces plateformes sont **coopératif-only**.
- csend ne **reconstruit pas** claude-mem (§1) : il **interopère** (journal exposé + lecture
  de contexte).
- Le réseau multi-machine n'est ouvert qu'en Phase 3 (surface réseau = coût sécurité §38).

## 9. Découpage en phases (livrable §30)

- **Phase 0** — fix self-bug + git/CI/Makefile/.nfo.
- **Phase 1** — backbone **inbox coopératif** + **registre/mémoire persistante** + **crypto base**
  (identités, E2E, vault).
- **Phase 2** — injection **tmux/screen** + adaptateurs **Codex/Gemini** + **passkey**.
- **Phase 3** — **réseau multi-machine** (daemon, TLS hybride PQC) + **recovery BIP-39+Shamir**.
- **Phase 4** — **clients mobiles** + **Windows** (coop) + **bridge Agent Teams** + interop
  claude-mem approfondie.
- **Phase 5** — durcissement PQC complet, audit crypto externe, autres OS.

Chaque phase tient debout seule ; la crypto arrive **dès la Phase 1**.

## 10. Stratégie de test

- **Détection d'état** : fixtures d'écrans **réels** capturés (§2), un par état/provider.
- **Transport inbox** : roundtrip write→read, persistance, ordre, idempotence.
- **Registre/mémoire** : CRUD + requêtes + cycles du graphe familial.
- **Crypto** : seal→open roundtrip, signature valide/invalide, vault lock→unlock, Shamir
  split→combine + seuil insuffisant rejeté.
- **Router** : table de décision (coop vs muet vs offline).
- **CI** : `go vet` + `go test ./...` + build universel à chaque push.

---

**Auteur** : Aïssa BELKOUSSA
