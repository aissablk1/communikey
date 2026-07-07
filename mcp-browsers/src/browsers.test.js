import { test } from "node:test";
import assert from "node:assert/strict";
import { browsers, findBrowser, listBrowsers } from "./browsers.js";

test("registre : intégrité des specs", () => {
  const ids = Object.keys(browsers);
  assert.ok(ids.length >= 10, "au moins 10 navigateurs catalogués");
  for (const [id, b] of Object.entries(browsers)) {
    assert.equal(b.id, id, `${id}: champ id cohérent avec la clé`);
    assert.ok(b.label, `${id}: label non vide`);
    assert.ok(["cdp", "none"].includes(b.controllable), `${id}: controllable valide`);
    assert.equal(typeof b.chromium, "boolean", `${id}: chromium booléen`);
    assert.equal(typeof b.verified, "boolean", `${id}: verified booléen`);
    assert.equal(b.aiRecipe, null, `${id}: aiRecipe null en Phase 1`);
    if (b.controllable === "cdp") {
      assert.ok(b.chromium, `${id}: pilotable CDP implique Chromium`);
      assert.ok(b.defaultPort > 0, `${id}: un port de debug > 0`);
    } else {
      assert.equal(b.defaultPort, 0, `${id}: non pilotable → port 0`);
    }
  }
});

test("ports de debug uniques par navigateur pilotable", () => {
  const ports = listBrowsers().filter((b) => b.controllable === "cdp").map((b) => b.defaultPort);
  assert.equal(ports.length, new Set(ports).size, "aucun port de debug dupliqué");
});

test("findBrowser", () => {
  assert.ok(findBrowser("dia"), "dia présent");
  assert.equal(findBrowser("navigateur-inexistant-xyz"), null, "inconnu → null");
});

test("listBrowsers trié et complet", () => {
  const l = listBrowsers();
  assert.equal(l.length, Object.keys(browsers).length);
  for (let i = 1; i < l.length; i++) {
    assert.ok(l[i - 1].id <= l[i].id, "ids triés");
  }
});
