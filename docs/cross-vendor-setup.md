# Câbler Claude Code + Codex + Gemini sur UN bus csend

Le pari défendable de csend : faire collaborer **plusieurs éditeurs d'agents** (Claude Code,
Codex CLI, Gemini CLI) dans un même bus local — la « seconde opinion » cross-vendor que les
orchestrateurs mono-éditeur ne donnent pas. Ce guide décrit le câblage **réel**.

> Honnêteté (§29) : la **réception coopérative** (hook) marche sur les trois éditeurs. L'**injection
> clavier live** (piloter une session sans qu'elle lise son inbox) est aujourd'hui fiable surtout
> côté Claude/tmux. Ce guide utilise la voie **coopérative** (hook), la plus portable.

## 1. Prérequis (honnêtes)

- **csend** installé (`go install github.com/aissablk1/csend@latest` ou binaire de release).
- **Claude Code** : rien de plus (hook `UserPromptSubmit`).
- **Codex CLI** : `npm i -g @openai/codex` ; Codex exige **d'approuver le hook** (`/hooks` ou
  `--dangerously-bypass-hook-trust`), sinon le message n'arrive jamais.
- **Gemini CLI** : installé **et authentifié** (`GEMINI_API_KEY` ou login OAuth) ; hook `BeforeAgent`.
- **Un store partagé** pour la voie coopérative locale : exporter le même `CSEND_STORE_DIR` dans les
  trois sessions (même machine). Pour des machines séparées : voir §5 (réseau).

## 2. Câbler la réception (hook) — une fois par session

Dans chaque session, afficher puis appliquer le snippet adapté à l'éditeur :

```sh
csend hook --install claude    # → snippet ~/.claude/settings.json (UserPromptSubmit)
csend hook --install codex     # → ~/.codex/ (+ APPROUVER le hook)
csend hook --install gemini    # → ~/.gemini/settings.json (BeforeAgent)
```

Le hook `csend hook` :
- dérive une **identité stable** du `session_id` passé sur stdin (zéro-config ; ou pose
  `CSEND_AGENT_ID` pour la forcer),
- draine l'inbox et **injecte les messages reçus dans le contexte** de la session,
- émet la **bonne forme** par éditeur (Claude/Codex : `hookSpecificOutput.additionalContext` ;
  Gemini : stdout brut). Force la forme avec `csend hook --provider {claude|codex|gemini}` si le
  CLI ne passe pas `hook_event_name`.

## 3. Rejoindre le bus (pour la visibilité `csend agents`)

```sh
CSEND_AGENT_ID=claude-dev csend register --provider claude
CSEND_AGENT_ID=codex-exec csend register --provider codex
CSEND_AGENT_ID=gemini-rev csend register --provider gemini
csend agents          # → les trois éditeurs sur un bus
```

## 4. Le motif « /second-opinion » (revue adversariale cross-vendor)

Claude écrit du code, puis demande une relecture **à froid** à un autre éditeur ; le verdict
**retombe dans la session vivante** via son hook, sans copier-coller :

```sh
# depuis Claude
csend inbox gemini-rev "relis ce diff et RÉFUTE-le si un cas limite casse (0,1,négatifs)"
# Gemini, à son prochain tour, voit la demande, relit, et répond :
csend inbox claude-dev "RÉFUTÉ : is_prime(1) doit renvoyer False — ajoute 'if n <= 1: return False'"
```

Le principe writer≠checker : demander à l'autre éditeur de **réfuter** (verdict ambigu = FAIL)
attrape des angles morts qu'un self-review d'un seul modèle rate.

## 5. Chiffrement E2E (optionnel) et machines séparées

Pour du **chiffré de bout en bout** (et entre machines), donner à chaque agent sa propre identité
et échanger les clés publiques :

```sh
export CSEND_VAULT_PASS=…            # déverrouille le vault
csend id --create                    # identité crypto locale
csend id --export                    # jeton public à partager
csend contact add <pair> <jeton>     # enregistre la clé publique d'un pair
# entre machines : le destinataire lance `csend serve`, l'émetteur `csend remote <hôte:port> <agent> <msg>`
```

`csend journal` montre alors la trace **de→à : sha256:… (chiffré)** — le corps n'apparaît jamais.

## 6. Démo

```sh
scripts/demo-cross-vendor.sh    # « Green Build Relay » : 3 éditeurs, 1 bus (relais réel)
```

> Limites (voir `docs/THREAT-MODEL.md`) : sur une même machine/UID, la crypto E2E apporte surtout
> du *defense-in-depth* ; l'agent récepteur reste une surface d'injection (défense au niveau du
> harnais, pas du bus) ; la crypto **n'est pas auditée**.

**Auteur** : Aïssa BELKOUSSA
