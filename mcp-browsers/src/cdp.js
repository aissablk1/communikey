// cdp.js — connecteur Chrome DevTools Protocol. Lance une app navigateur avec un
// port de debug, s'attache à une instance déjà lancée, détecte les cibles.
// Réutilise chrome-remote-interface (§1 — pas de client CDP maison).
//
// GARDE-FOU (§38/§44) : le port de debug est TOUJOURS lié à localhost. On ne se
// connecte JAMAIS à un host distant — quiconque atteint ce port pilote le
// navigateur et lit la session connectée de l'utilisateur.

import { spawn } from "node:child_process";
import { access } from "node:fs/promises";
import CDP from "chrome-remote-interface";

const HOST = "127.0.0.1"; // localhost STRICT — jamais paramétrable en distant (§38)

/** launchBrowser lance l'app avec --remote-debugging-port. Erreur explicite si le
 *  chemin est absent (navigateur non installé ou chemin à vérifier). */
export async function launchBrowser(spec, port) {
  if (spec.controllable !== "cdp") {
    throw new Error(`${spec.id} n'est pas pilotable par CDP (${spec.notes})`);
  }
  if (!spec.appPath) {
    throw new Error(`${spec.id} : aucun chemin d'app connu — lance-le manuellement avec --remote-debugging-port=${port}`);
  }
  try {
    await access(spec.appPath);
  } catch {
    throw new Error(
      `${spec.id} introuvable à ${spec.appPath}` +
      (spec.verified ? "" : " (chemin à vérifier §29)") +
      ` — installe-le, corrige le chemin, ou lance-le manuellement avec --remote-debugging-port=${port}`
    );
  }
  const child = spawn(spec.appPath, [`--remote-debugging-port=${port}`], {
    detached: true,
    stdio: "ignore",
  });
  child.unref();
  return { pid: child.pid, port };
}

/** attach renvoie un client CDP sur localhost:port (health-check implicite). */
export async function attach(port) {
  // chrome-remote-interface se lie à 'localhost' par défaut ; on force l'hôte.
  return CDP({ host: HOST, port });
}

/** listTargets renvoie les cibles ouvertes sur un port (page titles/urls), ou []
 *  si aucun navigateur n'écoute (jamais une exception qui casse browser_list). */
export async function listTargets(port) {
  try {
    const targets = await CDP.List({ host: HOST, port });
    return targets
      .filter((t) => t.type === "page")
      .map((t) => ({ title: t.title, url: t.url, id: t.id }));
  } catch {
    return [];
  }
}

/** isReachable teste si un port de debug répond en localhost. */
export async function isReachable(port) {
  try {
    await CDP.Version({ host: HOST, port });
    return true;
  } catch {
    return false;
  }
}
