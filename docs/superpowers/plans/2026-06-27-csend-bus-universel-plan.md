# csend v2 — Plan d'implémentation (phases)

Réf. design : `docs/superpowers/specs/2026-06-27-csend-bus-universel-design.md`.
Règle : TDD (test rouge → vert), `go vet` + `go test ./...` verts avant chaque commit (§32).

## Phase 0 — infra & sûreté  *(cette session)*
1. **Fix bug self** — `callerRef(tree)` privilégie `caller.surface_ref` (espace `surface:N`) sur
   l'UUID d'env ; `isSelf(surface, self)` = `Here || ref==self`. Tests unitaires.
   Câbler dans `deliverTo` + `resolveFamily`.
2. **Makefile** (build/test/vet/install) ; **CI** `.github/workflows/ci.yml` ; **PROJECT.nfo** (§24).

## Phase 1 — backbone universel  *(cette session, autant que possible)*
3. **`crypto.go`** — identité Ed25519+X25519, hybride ML-KEM (stdlib 1.24), seal/open, vault
   AES-256-GCM (clé PBKDF2/HKDF). Tests roundtrip.
4. **`memory.go`** — registre + journal append-only interrogeable (sessions, relations, messages).
   Tests CRUD/requêtes.
5. **`inbox.go`** — transport coopératif fichier (inbox par agent) + `csend recv`. Tests roundtrip.
6. **`router.go`** — décision inbox-d'abord / injection-repli. Tests de table.

## Stretch  *(si temps — sinon honnête, reporté)*
7. **`shamir.go`** GF(256) split/combine + seuil ; **BIP-39** mnémonique. Tests.
8. **backend tmux** (`tmux send-keys`) derrière le contrat transport.
9. **adaptateurs Codex/Gemini** (détection d'état) avec fixtures réelles.

## Phases ultérieures (hors session, documentées)
- Phase 2 passkey ; Phase 3 réseau multi-machine (TLS hybride PQC) ; Phase 4 mobile + Windows
  + bridge Agent Teams ; Phase 5 PQC complet + audit externe.

**Auteur** : Aïssa BELKOUSSA
