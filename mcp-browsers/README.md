# communikey-browsers — MCP de contrôle des navigateurs IA

Serveur **MCP** (stdio, Node/TS→ESM) qui pilote les **navigateurs dopés à l'IA**
(Dia, Perplexity Comet, ChatGPT Atlas, Brave Leo, Chrome/Gemini, Edge/Copilot,
Opera Neon, Arc, SigmaOS, Maxthon…) via le **Chrome DevTools Protocol**.
Sous-projet de [communikey](../). Design : `../docs/superpowers/specs/2026-07-08-communikey-browser-mcp-design.md`.

## Ce que ça apporte (vs playwright/chrome-devtools MCP, §1)

Le **registre des navigateurs IA** (`src/browsers.js`) : comment lancer chaque app
avec un port de debug, sa pilotabilité, et (Phase 2) comment invoquer **son IA
native**. Le pilotage bas niveau réutilise `chrome-remote-interface`.

## Installation

```bash
cd mcp-browsers
npm install
npm test          # tests du registre (sans navigateur)
```

Déclarer dans `~/.claude/settings.json` :

```json
{
  "mcpServers": {
    "communikey-browsers": {
      "command": "node",
      "args": ["/chemin/vers/communikey/mcp-browsers/src/index.js"]
    }
  }
}
```

Puis **redémarrer** le client MCP.

## Outils (Phase 1)

`browser_list` · `browser_launch {id}` · `browser_attach {id|port}` ·
`browser_navigate {url}` · `browser_read {selector?}` · `browser_click {selector}` ·
`browser_fill {selector,text}` · `browser_screenshot` · `browser_eval {js}`.

`browser_ai_ask {id,prompt}` (invoquer l'IA native) arrive en **Phase 2** (recettes
par navigateur, calibrées sur UI réelle).

## Usage type

```
browser_launch  {"id": "brave"}          # lance Brave avec --remote-debugging-port=9224
browser_attach  {"port": 9224}           # s'attache
browser_navigate {"url": "https://…"}
browser_read
```

Navigateur déjà lancé ? Lance-le manuellement avec le flag puis `browser_attach` :
`"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser" --remote-debugging-port=9224`

## Garde-fous (STRICT — §36/§38/§44)

- **Port de debug `localhost` uniquement, jamais exposé** au réseau. Quiconque
  atteint ce port pilote le navigateur et lit ta session connectée.
- Attacher = accès aux **sessions connectées** (comptes) → **usage personnel /
  recherche**. Les **CGU** d'OpenAI/Perplexity/Google **restreignent l'automation** :
  à toi de rester dans les clous. Pas d'automation de masse.
- `browser_eval` exécute du **JS arbitraire** dans un contexte authentifié — ne
  jamais l'utiliser pour exfiltrer secrets/PII.
- Certains navigateurs (Atlas, Comet) peuvent **verrouiller** le port de debug ;
  chemins d'app marqués `verified:false` = **à confirmer** sur la machine (§29).

## Statut

- **Phase 1** (ce commit) : registre + launch/attach/list + automation générique.
  Vérifié : build/import + tests du registre. Pilotage CDP live = **vérification
  manuelle** (nécessite un navigateur lancé avec le flag, comme tradingview-mcp).
- **Phase 2** : `browser_ai_ask` + recettes IA natives.
- **Phase 3** : intégration Go (`communikey browsers`) + `settings.json`.

**Auteur** : Aïssa BELKOUSSA
