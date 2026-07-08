package main

import "testing"

// relations.go n'avait AUCUN test automatisé avant celui-ci (audit du 2026-07-03,
// confirmé absent le 2026-07-05) — c'est le graphe familial qui gouverne send
// --up/--down/--to-siblings/--to-descendants et l'anti-cycle de `communikey link`.
// Les tests ci-dessous couvrent la logique de graphe en mémoire (aucune écriture
// disque), puis la persistance JSON séparément avec un HOME temporaire isolé.

func TestRelationsLinkReplacesExistingParent(t *testing.T) {
	r := &Relations{Names: map[string]string{}}
	r.link("A", "C") // A est parent de C
	r.link("B", "C") // C change de parent : B

	p, ok := r.parentOf("C")
	if !ok || p != "B" {
		t.Fatalf("parent de C = %q (ok=%v), attendu B", p, ok)
	}
	if len(r.Edges) != 1 {
		t.Fatalf("un ré-attachement doit REMPLACER l'arête, pas l'empiler : %d arêtes, attendu 1", len(r.Edges))
	}
}

func TestRelationsUnlinkChild(t *testing.T) {
	r := &Relations{Names: map[string]string{}}
	r.link("A", "B")
	r.unlinkChild("B")

	if _, ok := r.parentOf("B"); ok {
		t.Fatal("B ne devrait plus avoir de parent après unlink")
	}
	if len(r.Edges) != 0 {
		t.Fatalf("0 arête attendue après unlink, got %d", len(r.Edges))
	}

	// unlink d'un nœud déjà sans parent ne doit pas paniquer ni rien casser.
	r.unlinkChild("inconnu")
	if len(r.Edges) != 0 {
		t.Fatalf("unlink d'un nœud absent ne doit rien modifier, got %d arêtes", len(r.Edges))
	}
}

func TestRelationsChildrenOf(t *testing.T) {
	r := &Relations{Names: map[string]string{}}
	r.link("PARENT", "C1")
	r.link("PARENT", "C2")
	r.link("PARENT", "C3")
	r.link("AUTRE", "C4")

	kids := r.childrenOf("PARENT")
	if len(kids) != 3 {
		t.Fatalf("3 enfants attendus pour PARENT, got %d (%v)", len(kids), kids)
	}
	want := map[string]bool{"C1": true, "C2": true, "C3": true}
	for _, k := range kids {
		if !want[k] {
			t.Fatalf("enfant inattendu %q dans %v", k, kids)
		}
		delete(want, k)
	}
	if len(want) != 0 {
		t.Fatalf("enfants manquants: %v", want)
	}

	if kids := r.childrenOf("SANS-ENFANT"); len(kids) != 0 {
		t.Fatalf("aucun enfant attendu pour un nœud isolé, got %v", kids)
	}
}

func TestRelationsParentOf(t *testing.T) {
	r := &Relations{Names: map[string]string{}}
	r.link("A", "B")

	if p, ok := r.parentOf("B"); !ok || p != "A" {
		t.Fatalf("parent de B = %q (ok=%v), attendu A", p, ok)
	}
	if _, ok := r.parentOf("A"); ok {
		t.Fatal("A est racine, ne devrait avoir aucun parent")
	}
	if _, ok := r.parentOf("INCONNU"); ok {
		t.Fatal("un nœud jamais vu ne devrait avoir aucun parent")
	}
}

func TestRelationsWouldCycleSelfLink(t *testing.T) {
	r := &Relations{Names: map[string]string{}}
	// Un nœud ne peut pas être son propre parent — wouldCycle doit le détecter
	// directement, indépendamment du garde-fou explicite déjà posé dans cmdLink.
	if !r.wouldCycle("X", "X") {
		t.Fatal("wouldCycle(X, X) doit être vrai (auto-lien)")
	}
}

func TestRelationsWouldCycleDirect(t *testing.T) {
	r := &Relations{Names: map[string]string{}}
	r.link("A", "B") // A parent de B

	// Faire de B le parent de A créerait un cycle direct A→B→A.
	if !r.wouldCycle("B", "A") {
		t.Fatal("wouldCycle(B, A) doit être vrai : créerait un cycle direct A<->B")
	}
}

func TestRelationsWouldCycleIndirect(t *testing.T) {
	r := &Relations{Names: map[string]string{}}
	r.link("A", "B") // A parent de B
	r.link("B", "C") // B parent de C  → chaîne A→B→C

	// Faire de C le parent de A créerait un cycle indirect A→B→C→A.
	if !r.wouldCycle("C", "A") {
		t.Fatal("wouldCycle(C, A) doit être vrai : cycle indirect via la chaîne A→B→C")
	}
}

func TestRelationsWouldCycleFalseOnLegitimateReparenting(t *testing.T) {
	r := &Relations{Names: map[string]string{}}
	r.link("A", "B") // A parent de B
	// C est un nœud indépendant, hors de la lignée de B.

	// Faire de C le nouveau parent de B est un ré-attachement légitime, pas un cycle.
	if r.wouldCycle("C", "B") {
		t.Fatal("wouldCycle(C, B) doit être faux : C est hors de la lignée de B, ré-attachement légitime")
	}
}

// --- Persistance JSON (HOME temporaire isolé — ne touche jamais le vrai fichier
// de relations d'Aïssa dans ~/.claude/communikey/relations.json) ---

func TestRelationsSaveLoadRoundtrip(t *testing.T) {
	setTestHome(t, t.TempDir())

	r := loadRelations()
	r.link("PARENT", "ENFANT")
	r.Names["PARENT"] = "workspace-parent"
	r.Names["ENFANT"] = "workspace-enfant"
	if err := r.save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded := loadRelations()
	if p, ok := loaded.parentOf("ENFANT"); !ok || p != "PARENT" {
		t.Fatalf("après rechargement, parent de ENFANT = %q (ok=%v), attendu PARENT", p, ok)
	}
	if loaded.Names["PARENT"] != "workspace-parent" {
		t.Fatalf("nom du parent non persisté: %q", loaded.Names["PARENT"])
	}
}

func TestLoadRelationsNoFileYieldsEmpty(t *testing.T) {
	setTestHome(t, t.TempDir())

	r := loadRelations()
	if r == nil {
		t.Fatal("loadRelations ne doit jamais renvoyer nil")
	}
	if r.Names == nil {
		t.Fatal("Names doit être initialisé même sans fichier existant")
	}
	if len(r.Edges) != 0 {
		t.Fatalf("aucune arête attendue sans fichier existant, got %d", len(r.Edges))
	}
}
