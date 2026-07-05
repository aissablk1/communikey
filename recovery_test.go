package main

// recovery_test.go — couvre le point trouvé par l'audit sécurité du 2026-07-03 :
// `recovery combine` acceptait n'importe quelles ≥2 parts, sans checksum sur le
// secret reconstitué, et écrasait silencieusement identity.vault. Ces tests
// verrouillent le correctif : un checksum embarqué détecte une reconstruction
// invalide AVANT qu'elle n'atteigne UnmarshalIdentity/saveIdentity.

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestChecksummedSecretRoundtrip(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i * 7)
	}
	checked := checksummedSecret(secret)
	if len(checked) != len(secret)+checksumSuffixLen {
		t.Fatalf("longueur inattendue: %d", len(checked))
	}
	got, err := verifyChecksummedSecret(checked)
	if err != nil {
		t.Fatalf("checksum valide rejeté: %v", err)
	}
	if !bytes.Equal(got, secret) {
		t.Fatalf("secret altéré par l'aller-retour checksum")
	}
}

// TestShamirCombineWithChecksumRejectsWrongReconstruction reproduit le scénario
// de la faille : Shamir sous le seuil produit une valeur mathématiquement bien
// formée mais FAUSSE (propriété documentée dans shamir.go), et sans checksum,
// deriveIdentity l'acceptait quand même (toute graine 32 octets dérive une
// identité "valide"). Le checksum doit intercepter ce cas.
func TestShamirCombineWithChecksumRejectsWrongReconstruction(t *testing.T) {
	secret := make([]byte, 32)
	if _, err := readFull(secret); err != nil {
		t.Fatal(err)
	}
	checked := checksummedSecret(secret)

	shares, err := ShamirSplit(checked, 5 /*N*/, 3 /*K*/)
	if err != nil {
		t.Fatal(err)
	}

	// Seulement 2 parts sur un seuil de 3 : Shamir renvoie une valeur de la
	// bonne longueur, mais ce n'est PAS le secret (propriété du seuil).
	tooFew, err := ShamirCombine(shares[:2])
	if err != nil {
		t.Fatalf("ShamirCombine ne devrait pas échouer structurellement: %v", err)
	}
	if _, err := verifyChecksummedSecret(tooFew); err == nil {
		t.Fatal("le checksum aurait dû rejeter une reconstruction sous le seuil")
	}

	// Avec le seuil correct (K=3), la reconstruction doit passer le checksum
	// et redonner exactement le secret d'origine.
	enough, err := ShamirCombine(shares[:3])
	if err != nil {
		t.Fatal(err)
	}
	got, err := verifyChecksummedSecret(enough)
	if err != nil {
		t.Fatalf("checksum rejeté alors que le seuil est atteint: %v", err)
	}
	if !bytes.Equal(got, secret) {
		t.Fatal("secret reconstitué différent du secret d'origine")
	}
}

func TestVerifyChecksummedSecretRejectsTamperedChecksum(t *testing.T) {
	secret := make([]byte, 32)
	checked := checksummedSecret(secret)
	checked[len(checked)-1] ^= 0xFF // corrompt le dernier octet du checksum
	if _, err := verifyChecksummedSecret(checked); err == nil {
		t.Fatal("checksum corrompu accepté à tort")
	}
}

func TestVerifyChecksummedSecretRejectsShortInput(t *testing.T) {
	if _, err := verifyChecksummedSecret([]byte{1, 2, 3}); err == nil {
		t.Fatal("entrée trop courte acceptée à tort")
	}
}

func TestVaultExists(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Dir: dir}
	if vaultExists(s) {
		t.Fatal("vault signalé présent avant toute écriture")
	}
	if err := os.WriteFile(filepath.Join(dir, "identity.vault"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !vaultExists(s) {
		t.Fatal("vault non détecté après écriture")
	}
}

// readFull remplit b avec des octets non-nuls déterministes (pas besoin d'un
// vrai CSPRNG dans un test — juste éviter une graine entièrement à zéro qui
// masquerait un bug d'indexation).
func readFull(b []byte) (int, error) {
	for i := range b {
		b[i] = byte(i*31 + 13)
	}
	return len(b), nil
}
