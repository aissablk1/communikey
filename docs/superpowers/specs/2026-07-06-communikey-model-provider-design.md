# communikey — Client multi-provider de modèles (`model`) — Design

- **Projet** : communikey
- **Auteur** : Aïssa BELKOUSSA · contact@aissabelkoussa.fr
- **Date** : 2026-07-06
- **Statut** : design validé — implémentation par phases (ce doc couvre la Phase 1)
- **Tags** : model-provider, llm, ollama, localai, huggingface, gemini, cli, secrets

---

## 1. Problème & intention (vérifié)

communikey est aujourd'hui un bus de messages chiffré pour orchestrer des **sessions de CLI
d'agents déjà vivantes** (Claude Code, Codex, Gemini CLI…) — vérifié dans `provider.go` : son
seul sens de « provider » est un détecteur d'état par lecture d'écran
(`Detect(screen string) State`). Il n'appelle aujourd'hui **aucune** API de modèle de langage
(vérifié par grep sur le code : aucune occurrence d'endpoint Anthropic/OpenAI/Google, aucune
notion d'inférence).

Décision d'Aïssa (2026-07-06, brainstorming) : **étendre communikey** pour qu'il puisse aussi
**consommer directement des modèles** (locaux et cloud — Ollama, LocalAI, HuggingFace, Google
Gemini, Apple Foundation Models, etc.), avec trois usages cibles à terme :

1. **Commande utilitaire** — `communikey model call <name> "<prompt>"` (ce spec).
2. **Hooks enrichis** — `hook.go` utiliserait un modèle pour résumer/traduire/classer un
   message avant injection (hors scope de ce spec — Phase 2).
3. **Workers autonomes = nœuds du bus** — un « provider » de bus pourrait être un appel modèle
   direct plutôt qu'une session CLI à screen-scraper (hors scope de ce spec — Phase 3).

**Ce spec couvre uniquement la Phase 1** : le socle (client + config + secrets) et l'usage (1),
sur un périmètre volontairement restreint à 2 providers réels — c'est la base sur laquelle les
Phases 2 et 3 s'appuieront, chacune avec son propre design.

## 2. Corrections actées avant design (§29)

Deux points corrigés pendant le brainstorming, à ne pas réintroduire :

- **Liste crypto initialement proposée** (RSA-1024, SHA-1, MD5, algorithme de Shor, GNFS,
  Coppersmith–Winograd, Busy Beaver, classification des groupes finis simples, FairPlay,
  UPX+AES) : **hors sujet ou contre-productive** — primitives cassées, algorithmes d'attaque
  (pas de défense), théorie sans application, ou mauvais modèle de menace (DRM). **Aucun élément
  de cette liste n'entre dans ce design.** Ce qui est réutilisé est déjà en place : Ed25519,
  X25519+ML-KEM-768, AES-256-GCM, Argon2id/RFC 9106 (cf. `crypto.go`, `THREAT-MODEL.md`).
- **Portée initiale de la demande** (« tous les providers du marché ») : ramenée à un socle
  déclaratif extensible sans recompilation, validé de bout en bout sur **2 providers concrets**
  (un local, un cloud) plutôt que sur la liste complète d'un coup (§27 YAGNI).

## 3. Principe directeur

> **Tout est déclaratif dès le départ.** Contrairement au registre `provider.go` (socle Claude
> codé en dur + extensions JSON), il n'existe ici aucun détecteur « éprouvé sur écrans réels » à
> figer en code : même Ollama en local n'est qu'une URL par défaut. Ajouter un provider modèle
> compatible OpenAI = éditer `models.json`, jamais recompiler — même levier que
> `providers.json` (livré le 2026-07-05).

## 4. Architecture

Nouveaux fichiers (package `main`, cohérent avec le reste du binaire) :

```
modelprovider.go       interface ModelProvider + registre (construit depuis la config uniquement)
modelconfig.go         parsing de ~/.claude/communikey/models.json (miroir de providerconfig.go)
modelclient_openai.go  adaptateur générique "compatible OpenAI" (couvre Ollama + LocalAI en v1)
modelsecret.go         résolution d'un secret (vault existant | variable d'env)
model.go               cmdModel(args) : sous-commandes list | test | call
```

```go
// modelprovider.go
type ModelProvider interface {
    Name() string
    Complete(ctx context.Context, prompt string, opts ModelOptions) (string, error)
}

type ModelOptions struct {
    Model   string        // override du modèle par défaut du spec, optionnel
    Timeout time.Duration // défaut raisonnable si zéro
}
```

`main.go` : ajouter `case "model": cmdModel(args[1:])` dans le switch existant, au même niveau
que `provider`/`hook`/`journal`.

## 5. Composants

**`modelSpec` (JSON, `~/.claude/communikey/models.json`)** :

```json
{
  "models": [
    { "name": "ollama",     "kind": "openai-compatible", "base_url": "http://localhost:11434/v1", "model": "llama3.2" },
    { "name": "huggingface","kind": "openai-compatible", "base_url": "https://api-inference.huggingface.co/v1", "model": "…", "auth": "vault:huggingface" }
  ]
}
```

- `kind` : **seule valeur supportée en v1 : `"openai-compatible"`**. Tout autre `kind` déclaré
  est rejeté avec une erreur explicite à `model list` (jamais un échec silencieux, §29) — les
  formats natifs (HuggingFace Inference API hors mode compatible, pont Apple
  Foundation Models/MLX) sont des `kind` **futurs**, ajoutés seulement quand un vrai besoin les
  justifie (§27), pas anticipés ici.
- `auth` : soit `env:NOM_VAR`, soit `vault:<clé>` — résolu par `modelsecret.go`.
- Aucun provider n'est enregistré par défaut : **fichier absent = aucun provider** (même
  rétrocompatibilité zéro-config que `providers.json`). Aucun appel cloud n'est possible sans
  une entrée explicite — c'est le garde-fou opt-in validé avec Aïssa.

**Secrets (`modelsecret.go`)** — réutilise le vault existant (`SealVault`/`OpenVault` de
`crypto.go`, Argon2id → AES-256-GCM ; `resolveVaultPass()` de `bus.go`) plutôt qu'une nouvelle
crypto : un nouveau fichier scellé (même mécanisme, JSON `VaultBlob` distinct de celui de
l'identité) stocke les secrets de providers modèle, déverrouillé par le même mot de passe de
vault. `env:` reste supporté comme repli simple (moins sûr, signalé comme tel par `model list`,
jamais interdit).

> Correction du 2026-07-06 (§29) : la version initiale de ce document mentionnait PBKDF2-SHA256
> et `loadIdentity()`/`keyring.go` — le code a migré vers **Argon2id** (commit `eb73942`,
> concurrent à ce spec) et les primitives de scellement génériques (`SealVault`/`OpenVault`)
> vivent dans `crypto.go`, pas `keyring.go`. Corrigé pour refléter le code réel avant le plan
> d'implémentation.

**CLI (`model.go`)** :
- `communikey model list` — nom, `kind`, URL, secret présent/absent (jamais la valeur), miroir
  de `provider list`.
- `communikey model test <name>` — vérifie connectivité + auth via une **requête de complétion
  minimale** (1 jeton, prompt fixe type `"ping"`) — les endpoints compatibles OpenAI n'exposent
  pas de route de santé standard, donc pas de branche alternative à prévoir. Miroir de
  `provider test`.
- `communikey model call <name> "<prompt>"` (ou `--stdin`) — génération réelle. `--json` pour
  sortie structurée (même convention que `journal --json`). `--timeout <durée>` optionnel.

## 6. Flux de données

```
model call ollama "résume ce texte"
  → lecture de models.json (erreur explicite si JSON invalide)
  → spec "ollama" trouvée, kind=openai-compatible
  → résolution de auth (aucune ici, Ollama local sans clé)
  → POST {base_url}/chat/completions {model, messages:[{role:user, content:prompt}]}
  → parsing de la réponse (choices[0].message.content)
  → stdout (texte, ou JSON si --json)
```

Toute erreur (réseau, HTTP non-2xx, JSON malformé, timeout) remonte via `fail()` (helper déjà
utilisé partout dans le projet) avec un message explicite — jamais une chaîne vide qui
ressemblerait à une réponse valide.

## 7. Sécurité

- **Aucun appel vers un provider non déclaré explicitement** — pas de provider cloud par défaut,
  conforme à la décision « local + cloud dès le départ, mais opt-in par provider ».
- **Secret jamais loggé, jamais affiché** par `model list`/`model test` (statut seulement).
- **Vault réutilisé, pas de nouvelle primitive crypto** — cohérent avec `THREAT-MODEL.md` (pas
  d'audit crypto externe à ce jour ; ne pas complexifier la surface auditée).
- **`--timeout` oblig/défaut raisonnable** : un provider local down (Ollama non lancé) ou un
  provider cloud lent ne doit jamais bloquer indéfiniment.

## 8. Non-goals / limites honnêtes (Phase 1)

- **Pas de streaming** — réponse complète uniquement en v1 ; le streaming est un ajout naturel
  plus tard si un besoin réel apparaît (§27).
- **Un seul `kind` supporté** (`openai-compatible`) — HuggingFace Inference API en mode natif,
  Google Gemini en mode natif (si son mode compatible OpenAI ne convient pas), et un pont Apple
  Foundation Models (Swift/macOS-only, ne peut pas être un simple client HTTP Go générique) sont
  des `kind` futurs, ajoutés à la demande, pas anticipés.
- **Pas de hooks enrichis, pas de workers-comme-nœuds-du-bus** — Phases 2 et 3, chacune sa
  propre spec, une fois ce socle validé en usage réel.
- **Aucune garantie de couverture de « tout le marché »** — le socle couvre nativement tout
  endpoint réellement compatible OpenAI (Ollama, LocalAI, et d'autres selon déclaration) ; les
  providers à format propriétaire ne sont couverts qu'un par un, sur demande.

## 9. Découpage en phases

- **Phase 1** (ce spec) — `ModelProvider` + `models.json` + adaptateur OpenAI-compatible +
  secrets vault + `model list`/`test`/`call`, validés sur **Ollama (local)** et **HuggingFace
  Inference API (cloud)** — choisi comme premier cloud car sa documentation d'un mode compatible
  OpenAI est directement actionnable ; **Gemini reste l'alternative documentée** si le mode
  compatible OpenAI de HuggingFace s'avère insuffisant à l'implémentation (à vérifier sur la
  doc officielle à ce moment-là, §29 — ne pas présumer le détail d'API avant de coder).
- **Phase 2** — hooks enrichis (`hook.go` utilise un `ModelProvider` pour résumer/traduire/
  classer avant injection). Design séparé.
- **Phase 3** — workers autonomes comme nœuds du bus (un `ModelProvider` peut devenir un pair du
  registre `agents`/`teams`). Design séparé, dépend du bridge Agent Teams déjà partiellement
  bloqué (cf. vision 2026-07-05).

## 10. Stratégie de test

- **Config** : parsing JSON valide/invalide (`modelconfig_test.go`), `kind` inconnu rejeté
  explicitement.
- **Adaptateur OpenAI-compatible** : `httptest.Server` (stdlib Go, zéro dépendance ajoutée) qui
  simule une réponse `chat/completions` — jamais un vrai appel réseau dans les tests (même
  esprit que `provider_test.go`/`adapters_test.go`).
- **Secrets** : résolution `env:`/`vault:` testée avec un vault de test (déjà présent dans les
  helpers de tests crypto existants).
- **CLI** : `model list`/`test`/`call` — arguments valides/invalides, jamais de panique sur une
  entrée malformée.
- **Vérification manuelle réelle** (pas en CI, comme `provider test`) : un vrai Ollama local +
  une vraie clé du provider cloud choisi, via `model test`.

---

**Auteur** : Aïssa BELKOUSSA
