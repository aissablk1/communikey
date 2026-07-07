// tools.js — surface MCP : définitions d'outils + handlers. Logique séparée de
// l'index (transport) pour être testable. Chaque handler renvoie un résultat MCP
// { content: [...] } ; toute erreur est levée (index.js la mappe en isError).

import { findBrowser, listBrowsers } from "./browsers.js";
import { launchBrowser, attach, listTargets, isReachable } from "./cdp.js";

// Sessions CDP réutilisées par port + port "courant" (dernier attaché).
const sessions = new Map(); // port -> client chrome-remote-interface
let currentPort = null;

async function getSession(port) {
  if (port == null) {
    if (currentPort == null) {
      throw new Error("aucun navigateur attaché — utilise d'abord browser_attach {id|port}");
    }
    port = currentPort;
  }
  let client = sessions.get(port);
  if (client) return client;
  client = await attach(port);
  await client.Page.enable();
  await client.Runtime.enable();
  sessions.set(port, client);
  currentPort = port;
  return client;
}

const text = (s) => ({ content: [{ type: "text", text: s }] });

// evalInPage exécute une expression JS dans la page et renvoie sa valeur.
async function evalInPage(client, expression) {
  const { result, exceptionDetails } = await client.Runtime.evaluate({
    expression,
    returnByValue: true,
    awaitPromise: true,
  });
  if (exceptionDetails) {
    throw new Error(exceptionDetails.exception?.description || exceptionDetails.text || "erreur JS dans la page");
  }
  return result.value;
}

export const TOOLS = [
  {
    name: "browser_list",
    description:
      "Liste le catalogue des navigateurs IA (id, pilotabilité, port de debug par défaut) et les instances détectées comme joignables en localhost.",
    inputSchema: { type: "object", properties: {}, additionalProperties: false },
  },
  {
    name: "browser_launch",
    description:
      "Lance (ou refocalise) l'app navigateur avec un port de debug CDP. Nécessite le chemin de l'app (voir browser_list). Sinon, lance l'app manuellement avec --remote-debugging-port=<port> puis browser_attach.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "id du navigateur (voir browser_list)" },
        port: { type: "number", description: "port de debug (défaut : celui du catalogue)" },
      },
      required: ["id"], additionalProperties: false,
    },
  },
  {
    name: "browser_attach",
    description: "S'attache à un navigateur déjà lancé avec un port de debug (localhost strict) et le définit comme navigateur courant.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "id du navigateur (utilise son port par défaut)" },
        port: { type: "number", description: "port de debug explicite (prioritaire sur id)" },
      },
      additionalProperties: false,
    },
  },
  {
    name: "browser_navigate",
    description: "Navigue vers une URL dans le navigateur courant (ou celui du port fourni).",
    inputSchema: {
      type: "object",
      properties: { url: { type: "string" }, port: { type: "number" } },
      required: ["url"], additionalProperties: false,
    },
  },
  {
    name: "browser_read",
    description: "Renvoie le texte de la page (ou d'un sélecteur CSS si fourni).",
    inputSchema: {
      type: "object",
      properties: { selector: { type: "string" }, port: { type: "number" } },
      additionalProperties: false,
    },
  },
  {
    name: "browser_click",
    description: "Clique sur le premier élément correspondant au sélecteur CSS.",
    inputSchema: {
      type: "object",
      properties: { selector: { type: "string" }, port: { type: "number" } },
      required: ["selector"], additionalProperties: false,
    },
  },
  {
    name: "browser_fill",
    description: "Écrit un texte dans un champ (sélecteur CSS) et déclenche les événements input/change.",
    inputSchema: {
      type: "object",
      properties: { selector: { type: "string" }, text: { type: "string" }, port: { type: "number" } },
      required: ["selector", "text"], additionalProperties: false,
    },
  },
  {
    name: "browser_screenshot",
    description: "Capture la page courante en PNG (base64).",
    inputSchema: {
      type: "object",
      properties: { port: { type: "number" } }, additionalProperties: false,
    },
  },
  {
    name: "browser_eval",
    description:
      "Exécute du JavaScript arbitraire dans la page (outil puissant — contexte AUTHENTIFIÉ : ne pas exfiltrer de secrets/PII).",
    inputSchema: {
      type: "object",
      properties: { js: { type: "string" }, port: { type: "number" } },
      required: ["js"], additionalProperties: false,
    },
  },
];

export async function handleTool(name, args = {}) {
  switch (name) {
    case "browser_list": {
      const catalog = listBrowsers().map((b) => ({
        id: b.id, label: b.label, controllable: b.controllable,
        defaultPort: b.defaultPort, chromium: b.chromium, verified: b.verified,
        notes: b.notes,
      }));
      const detected = [];
      // Scanne les ports du catalogue (localhost) sans jamais échouer.
      const seen = new Set();
      for (const b of listBrowsers()) {
        if (b.defaultPort > 0 && !seen.has(b.defaultPort)) {
          seen.add(b.defaultPort);
          if (await isReachable(b.defaultPort)) {
            detected.push({ port: b.defaultPort, targets: await listTargets(b.defaultPort) });
          }
        }
      }
      return { content: [{ type: "text", text: JSON.stringify({ catalog, detected }, null, 2) }] };
    }

    case "browser_launch": {
      const spec = findBrowser(args.id);
      if (!spec) throw new Error(`navigateur ${JSON.stringify(args.id)} absent du catalogue (voir browser_list)`);
      const port = args.port ?? spec.defaultPort;
      const { pid, port: p } = await launchBrowser(spec, port);
      return text(`✓ ${spec.label} lancé (pid ${pid}) avec --remote-debugging-port=${p}. Attache-toi : browser_attach {"port": ${p}}`);
    }

    case "browser_attach": {
      let port = args.port;
      if (port == null && args.id) {
        const spec = findBrowser(args.id);
        if (!spec) throw new Error(`navigateur ${JSON.stringify(args.id)} absent du catalogue`);
        if (spec.controllable !== "cdp") throw new Error(`${spec.id} n'est pas pilotable par CDP (${spec.notes})`);
        port = spec.defaultPort;
      }
      if (port == null) throw new Error("fournis un id ou un port");
      if (!(await isReachable(port))) {
        throw new Error(`aucun navigateur joignable sur localhost:${port} — lance-le d'abord (browser_launch <id>) ou avec --remote-debugging-port=${port}`);
      }
      await getSession(port);
      const targets = await listTargets(port);
      return { content: [{ type: "text", text: JSON.stringify({ attached: port, targets }, null, 2) }] };
    }

    case "browser_navigate": {
      const client = await getSession(args.port);
      await client.Page.navigate({ url: args.url });
      await client.Page.loadEventFired();
      return text(`✓ navigué vers ${args.url}`);
    }

    case "browser_read": {
      const client = await getSession(args.port);
      const expr = args.selector
        ? `(document.querySelector(${JSON.stringify(args.selector)})?.innerText ?? "")`
        : `document.body.innerText`;
      const value = await evalInPage(client, expr);
      return text(String(value ?? ""));
    }

    case "browser_click": {
      const client = await getSession(args.port);
      const sel = JSON.stringify(args.selector);
      const ok = await evalInPage(client, `(() => { const el = document.querySelector(${sel}); if (!el) return false; el.click(); return true; })()`);
      if (!ok) throw new Error(`aucun élément pour le sélecteur ${args.selector}`);
      return text(`✓ clic sur ${args.selector}`);
    }

    case "browser_fill": {
      const client = await getSession(args.port);
      const sel = JSON.stringify(args.selector);
      const val = JSON.stringify(args.text);
      const ok = await evalInPage(client, `(() => { const el = document.querySelector(${sel}); if (!el) return false; el.value = ${val}; el.dispatchEvent(new Event('input',{bubbles:true})); el.dispatchEvent(new Event('change',{bubbles:true})); return true; })()`);
      if (!ok) throw new Error(`aucun champ pour le sélecteur ${args.selector}`);
      return text(`✓ champ ${args.selector} rempli`);
    }

    case "browser_screenshot": {
      const client = await getSession(args.port);
      const { data } = await client.Page.captureScreenshot({ format: "png" });
      return { content: [{ type: "image", data, mimeType: "image/png" }] };
    }

    case "browser_eval": {
      const client = await getSession(args.port);
      const value = await evalInPage(client, args.js);
      return text(typeof value === "string" ? value : JSON.stringify(value));
    }

    default:
      throw new Error(`outil inconnu: ${name}`);
  }
}
