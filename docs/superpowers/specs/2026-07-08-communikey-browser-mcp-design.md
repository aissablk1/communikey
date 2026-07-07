# communikey-browsers — MCP de contrôle des navigateurs IA — Design

> Statut : design validé (Aïssa, 2026-07-08). Implémentation par phases.
> **Auteur** : Aïssa BELKOUSSA

## 1. Objectif

Donner à communikey un **serveur MCP** (`communikey-browsers`) qui pilote les
**navigateurs dopés à l'IA** (Dia, Perplexity Comet, ChatGPT Atlas, Brave Leo,
Chrome/Gemini, Edge/Copilot, Opera Neon, Arc, SigmaOS, Maxthon…) via le **Chrome
DevTools Protocol** (CDP). Le différenciateur vs les MCP d'automatisation déjà
présents (`playwright`, `chrome-devtools`, `scrapling`, §1) est le **registre des
navigateurs IA** — comment lancer chaque app avec un port de debug et comment
invoquer **son IA native** — pas le pilotage bas niveau d'une page (réutilisé).

## 2. Stack & emplacement

- **Node/TS**, sous-projet `mcp-browsers/` dans le repo communikey (précédent :
  `tradingview-mcp` §44, même mécanisme CDP). Le binaire **Go communikey reste
  inchangé** (ethos zéro-dépendance préservé).
- Dépendances : `@modelcontextprotocol/sdk` (serveur MCP stdio),
  `chrome-remote-interface` (client CDP). Aucune donnée ne quitte la machine.
- Déclaré dans `~/.claude/settings.json` → `mcpServers.communikey-browsers`.

## 3. Architecture

```
agent ⇄ MCP stdio (Node/TS) ⇄ CDP (localhost:PORT) ⇄ Navigateur IA (Chromium)
              │
              └─ REGISTRE (browsers.ts) — catalogue vérifié des navigateurs IA
```

Trois unités isolées, chacune testable indépendamment :

1. **`browsers.ts` — le registre.** Catalogue `id → BrowserSpec` :
   `label`, `appPath` (chemin macOS, `[à vérifier]` si non confirmé §29),
   `bundleId`, `defaultPort`, `chromium: boolean`, `controllable: "cdp" | "none"`,
   `aiRecipe` (recette d'invocation de l'IA native — `null` en Phase 1),
   `notes` (CGU / limites). Fonctions pures : `findBrowser(id)`, `listBrowsers()`.
2. **`cdp.ts` — le connecteur.** `launch(spec, port)` (spawn l'app avec
   `--remote-debugging-port`), `attach(port)` (chrome-remote-interface + health),
   `detectRunning()` (scan des ports de debug ouverts en localhost). Ne pilote pas
   l'IA ; expose une session CDP.
3. **`tools.ts` + `index.ts` — la surface MCP.** Enregistre les outils, valide les
   entrées, mappe erreurs → messages explicites (jamais un échec silencieux §29).

## 4. Outils MCP

| Outil | Rôle | Phase |
|---|---|---|
| `browser_list` | catalogue + instances détectées (running/attachable) | 1 |
| `browser_launch {id, port?}` | lance/focus l'app avec port de debug | 1 |
| `browser_attach {id?|port}` | attache CDP + health-check | 1 |
| `browser_navigate {url}` | va à une URL | 1 |
| `browser_read {selector?}` | texte/DOM de la page (ou d'un sélecteur) | 1 |
| `browser_click {selector}` · `browser_fill {selector, text}` | interaction | 1 |
| `browser_screenshot` | capture PNG (base64) | 1 |
| `browser_eval {js}` | exécute du JS dans la page (outil puissant, documenté) | 1 |
| `browser_ai_ask {id, prompt}` | **invoque l'IA native** du navigateur, renvoie sa réponse | 2 |

## 5. Flux de données

`browser_ai_ask` (Phase 2) : attach → applique la **recette** du navigateur
(ouvre le panneau IA, écrit le prompt, soumet, attend, lit la réponse via
sélecteurs/CDP) → renvoie le texte. Chaque recette est **calibrée sur l'UI réelle**
du navigateur (§2/§29), jamais inventée, et marquée fragile.

## 6. Gestion d'erreurs

- Navigateur non lancé / port fermé → message explicite (« lance-le avec
  `browser_launch <id>` ou `--remote-debugging-port=<port>` »).
- `id` absent du registre → suggère `browser_list`.
- `controllable: "none"` (Vivaldi, Ladybird…) → refus explicite, jamais un faux OK.
- Recette IA cassée (UI changée) → erreur claire, pas une réponse vide crédible.

## 7. Sécurité (garde-fous STRICT — §36/§38/§44)

- **Port de debug `localhost` uniquement, jamais `0.0.0.0`** ni exposé réseau.
- Attacher un navigateur = accès aux **sessions connectées** de l'utilisateur →
  **usage personnel/recherche**, avertir sur les **CGU** de chaque service
  (OpenAI/Perplexity/Google restreignent l'automation) ; pas d'automation de masse.
- `browser_eval` = exécution de JS arbitraire dans un contexte authentifié →
  documenté comme outil puissant, jamais utilisé pour exfiltrer secrets/PII.
- Aucun secret manipulé/loggué ; ports fermés après usage.

## 8. Couverture (honnête, §29)

- ✅ **Pilotables (cdp)** : Chrome, Edge, Brave, Arc, Opera Neon, Maxthon, SigmaOS,
  Dia, Comet, Atlas *(si le port de debug n'est pas verrouillé — à vérifier)*, Fellou.
- ⚠️ **`browser_ai_ask` par navigateur** : recette fragile, calibrée, incrémentale.
- ❌ **Non pilotables** (listés, `controllable: "none"`) : Vivaldi (anti-IA),
  Ladybird (pas de CDP), natifs sans port de debug, Opera Air.

## 9. Non-goals

Pas de gestion de profils/comptes, pas d'automation de masse, pas de contournement
de CGU, pas de pilotage des versions mobiles, pas de réécriture d'un moteur CDP
(on réutilise chrome-remote-interface, §1).

## 10. Phases

- **Phase 1** : registre + `launch`/`attach`/`list` + automation générique
  (navigate/read/click/fill/screenshot/eval) → contrôle immédiat de **tous** les
  Chromium. Testable sans IA native.
- **Phase 2** : `browser_ai_ask` + recettes IA natives (Dia, Comet, Brave Leo,
  Atlas…), calibrées sur UI réelle, ajoutées une par une.
- **Phase 3** : intégration Go (`communikey browsers` lance/déclare le MCP) +
  entrée `settings.json` + doc.

## 11. Tests

- Registre : intégrité (ids uniques, `controllable ∈ {cdp,none}`, ports valides,
  `chromium` cohérent) — pur, sans navigateur.
- Outils : validation des entrées + mapping d'erreurs (sans navigateur, via mocks
  de session CDP au niveau interface).
- CDP live : vérification **manuelle** documentée (nécessite un navigateur lancé
  avec `--remote-debugging-port`, comme tradingview-mcp §44).
