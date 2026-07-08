// browsers.js — registre des navigateurs IA (le cœur "communikey" du MCP, pattern
// des catalogues provider/model du binaire Go). Chaque entrée décrit comment
// LANCER l'app avec un port de debug CDP et (Phase 2) comment invoquer SON IA
// native. Le pilotage bas niveau réutilise chrome-remote-interface (§1).
//
// PROVENANCE / HONNÊTETÉ (§29) :
//   • verified:true  = chemin macOS confiant (navigateur répandu).
//   • verified:false = chemin PROBABLE, à confirmer sur la machine (navigateur
//     récent) — `browser_attach {port}` fonctionne de toute façon SANS le chemin
//     dès que l'app tourne avec --remote-debugging-port.
//   • controllable: "cdp"  = Chromium pilotable via CDP.
//     controllable: "none" = pas de CDP (non-Chromium, ou natif sans port debug).
//   • aiRecipe: null en Phase 1 (les recettes d'IA native arrivent en Phase 2).
//
// Le flag de debug est UNIVERSEL pour Chromium : --remote-debugging-port=<port>.

/** @typedef {{
 *   id: string, label: string, appPath: string|null, bundleId: string|null,
 *   defaultPort: number, chromium: boolean, controllable: "cdp"|"none",
 *   verified: boolean, aiRecipe: null, notes: string
 * }} BrowserSpec */

const mac = (app, bin) => `/Applications/${app}.app/Contents/MacOS/${bin ?? app}`;

/** @type {Record<string, BrowserSpec>} */
export const browsers = {
  // ── Classiques dopés à l'IA (chemins confiants) ────────────────────────────
  chrome: {
    id: "chrome", label: "Google Chrome (Gemini)", appPath: mac("Google Chrome"),
    bundleId: "com.google.Chrome", defaultPort: 9222, chromium: true,
    controllable: "cdp", verified: true, aiRecipe: null,
    notes: "Gemini en panneau latéral. Debug port standard.",
  },
  edge: {
    id: "edge", label: "Microsoft Edge (Copilot)", appPath: mac("Microsoft Edge"),
    bundleId: "com.microsoft.edgemac", defaultPort: 9223, chromium: true,
    controllable: "cdp", verified: true, aiRecipe: null,
    notes: "Copilot Mode / Agent Mode.",
  },
  brave: {
    id: "brave", label: "Brave (Leo AI)", appPath: mac("Brave Browser"),
    bundleId: "com.brave.Browser", defaultPort: 9224, chromium: true,
    controllable: "cdp", verified: true, aiRecipe: null,
    notes: "Leo AI (choix du modèle, BYOM via Ollama). Le plus orienté vie privée.",
  },
  arc: {
    id: "arc", label: "Arc (Max)", appPath: mac("Arc"),
    bundleId: "company.thebrowser.Browser", defaultPort: 9225, chromium: true,
    controllable: "cdp", verified: true, aiRecipe: null,
    notes: "Arc Max (résumés, rangement d'onglets). En mode maintenance (remplacé par Dia).",
  },
  vivaldi: {
    id: "vivaldi", label: "Vivaldi (sans IA)", appPath: mac("Vivaldi"),
    bundleId: "com.vivaldi.Vivaldi", defaultPort: 9226, chromium: true,
    controllable: "cdp", verified: true, aiRecipe: null,
    notes: "Chromium PILOTABLE (page), mais REFUSE toute IA native → pas de browser_ai_ask.",
  },

  // ── 100% IA (chemins à confirmer sur la machine — §29) ──────────────────────
  dia: {
    id: "dia", label: "Dia (The Browser Company)", appPath: mac("Dia"),
    bundleId: "company.thebrowser.dia", defaultPort: 9230, chromium: true,
    controllable: "cdp", verified: true, aiRecipe: null,
    notes: "IA conversationnelle native, Skills, mémoire 7 jours. macOS Apple Silicon. Chemin + bundle id vérifiés le 2026-07-08. Port de debug CDP confirmé ACTIF (sonde 2026-07-08) → launch/attach OK ; MAIS en profil neuf : 1 cible 'other', 0 'page' → l'UI IA n'est pas une cible page standard → browser_ai_ask (Phase 2) nécessite la vraie session Dia connectée + déterminer la cible de l'IA (page vs native).",
  },
  comet: {
    id: "comet", label: "Perplexity Comet", appPath: mac("Comet"),
    bundleId: "ai.perplexity.comet", defaultPort: 9231, chromium: true,
    controllable: "cdp", verified: false, aiRecipe: null,
    notes: "Barre d'URL = barre de prompt, navigation agentique. Chemin/port debug à vérifier.",
  },
  atlas: {
    id: "atlas", label: "ChatGPT Atlas (OpenAI)", appPath: mac("ChatGPT Atlas"),
    bundleId: "com.openai.atlas", defaultPort: 9232, chromium: true,
    controllable: "cdp", verified: false, aiRecipe: null,
    notes: "ChatGPT par onglet, mode Agent. macOS. ATTENTION : le port de debug peut être verrouillé.",
  },
  "opera-neon": {
    id: "opera-neon", label: "Opera Neon (agentique)", appPath: mac("Opera Neon"),
    bundleId: "com.opera.Neon", defaultPort: 9233, chromium: true,
    controllable: "cdp", verified: false, aiRecipe: null,
    notes: "Version agentique payante d'Opera. Chemin à vérifier.",
  },
  sigmaos: {
    id: "sigmaos", label: "SigmaOS (Airis)", appPath: mac("SigmaOS"),
    bundleId: "com.sigmaos.sigmaos.macos", defaultPort: 9234, chromium: true,
    controllable: "cdp", verified: false, aiRecipe: null,
    notes: "Mac only, productivité (workspaces, onglets verticaux) + assistant Airis. À vérifier.",
  },
  maxthon: {
    id: "maxthon", label: "Maxthon", appPath: mac("Maxthon"),
    bundleId: "com.maxthon.mac.Maxthon", defaultPort: 9235, chromium: true,
    controllable: "cdp", verified: false, aiRecipe: null,
    notes: "Navigation cloud + outils IA. À vérifier.",
  },
  fellou: {
    id: "fellou", label: "Fellou (autonome)", appPath: mac("Fellou"),
    bundleId: null, defaultPort: 9236, chromium: true,
    controllable: "cdp", verified: false, aiRecipe: null,
    notes: "Navigateur autonome pour workflows multi-étapes. À vérifier.",
  },

  // ── Non pilotables (listés honnêtement — §29) ───────────────────────────────
  ladybird: {
    id: "ladybird", label: "Ladybird", appPath: null, bundleId: null,
    defaultPort: 0, chromium: false, controllable: "none", verified: false,
    aiRecipe: null, notes: "Moteur from scratch (alpha 2026), PAS de CDP.",
  },
  duckduckgo: {
    id: "duckduckgo", label: "DuckDuckGo", appPath: null,
    bundleId: "com.duckduckgo.macos.browser", defaultPort: 0, chromium: false,
    controllable: "none", verified: false, aiRecipe: null,
    notes: "App macOS WebKit (pas Chromium) → pas de port de debug CDP.",
  },
};

/** findBrowser renvoie le spec, ou null. */
export function findBrowser(id) {
  return Object.prototype.hasOwnProperty.call(browsers, id) ? browsers[id] : null;
}

/** listBrowsers renvoie les specs triés par id (ordre stable). */
export function listBrowsers() {
  return Object.keys(browsers).sort().map((id) => browsers[id]);
}
