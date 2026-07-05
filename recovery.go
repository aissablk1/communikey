package main

// recovery.go — CLI de recovery par parts Shamir sur l'identité du vault.
//
//	communikey recovery split <K> <N>            découpe le secret d'identité en N parts (K-sur-N)
//	communikey recovery combine [--force] <p…>   reconstitue l'identité depuis ≥ K parts
//
// Le secret découpé est la graine maître 32 octets (sign+x25519+mlkem en dérivent) +
// un checksum de 4 octets (sha256 tronqué, cf. checksummedSecret). Shamir sous le
// seuil produit une valeur mathématiquement bien formée mais FAUSSE (propriété du
// schéma, voir shamir.go) — sans ce checksum, n'importe quelle graine de 32 octets
// dérive une identité "valide" en apparence, et `combine` écrasait silencieusement
// le vault sur une reconstruction bidon (trouvé par l'audit sécurité du 2026-07-03).
// Le checksum détecte ce cas AVANT dérivation ; `--force` reste requis pour écraser
// un vault déjà présent, même avec un checksum valide.
//
// Répartir les parts (téléphone, YubiKey, proche, papier coffre…) : perte d'un
// device ≠ perte du vault, et aucune part isolée ne révèle rien.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// checksumSuffixLen est le nombre d'octets de checksum ajoutés à la graine
// maître (32 octets) avant le découpage Shamir.
const checksumSuffixLen = 4

// checksummedSecret ajoute un checksum sha256 tronqué au secret, pour que
// `combine` puisse détecter une reconstruction invalide (parts insuffisantes,
// incorrectes, ou mauvais seuil K) au lieu de dériver une identité au hasard.
func checksummedSecret(secret []byte) []byte {
	sum := sha256.Sum256(secret)
	out := make([]byte, 0, len(secret)+checksumSuffixLen)
	out = append(out, secret...)
	out = append(out, sum[:checksumSuffixLen]...)
	return out
}

// verifyChecksummedSecret sépare le secret de son checksum et refuse tout ce
// qui ne correspond pas — c'est ce qui rend `combine` sûr à auto-écrire au
// lieu de deviner.
func verifyChecksummedSecret(checked []byte) ([]byte, error) {
	if len(checked) <= checksumSuffixLen {
		return nil, errors.New("secret reconstruit trop court")
	}
	cut := len(checked) - checksumSuffixLen
	secret, gotSum := checked[:cut], checked[cut:]
	wantSum := sha256.Sum256(secret)
	if !bytes.Equal(gotSum, wantSum[:checksumSuffixLen]) {
		return nil, errors.New("checksum invalide — parts insuffisantes, incorrectes, ou mauvais seuil K (aucune identité écrasée)")
	}
	return secret, nil
}

// vaultExists indique si un vault d'identité est déjà présent — utilisé pour
// exiger --force avant de l'écraser depuis une recovery.
func vaultExists(s *Store) bool {
	_, err := os.Stat(identityVaultPath(s))
	return err == nil
}

func loadIdentity(s *Store, pass []byte) (*Identity, error) {
	data, err := os.ReadFile(identityVaultPath(s))
	if err != nil {
		return nil, fmt.Errorf("aucun vault d'identité (communikey id --create d'abord): %w", err)
	}
	var blob VaultBlob
	if err := json.Unmarshal(data, &blob); err != nil {
		return nil, err
	}
	secret, err := OpenVault(&blob, pass)
	if err != nil {
		return nil, err
	}
	return UnmarshalIdentity(secret)
}

func cmdRecovery(args []string) {
	if len(args) < 1 {
		fail("usage: communikey recovery split <K> <N>  |  communikey recovery combine [--force] <part…>  |  communikey recovery phrase  |  communikey recovery from-phrase [--force] \"<mots>\"")
	}
	s := mustStore()
	switch args[0] {
	case "split":
		if len(args) < 3 {
			fail("usage: communikey recovery split <K> <N>  (K = seuil, N = nb de parts)")
		}
		k, err1 := strconv.Atoi(args[1])
		n, err2 := strconv.Atoi(args[2])
		if err1 != nil || err2 != nil {
			fail("K et N doivent être des entiers")
		}
		pass, ok := resolveVaultPass()
		if !ok {
			fail("définis COMKEY_VAULT_PASS_FILE ou COMKEY_VAULT_PASS pour ouvrir le vault à découper")
		}
		id, err := loadIdentity(s, pass)
		if err != nil {
			fail(err.Error())
		}
		secret, err := id.MarshalSecret()
		if err != nil {
			fail(err.Error())
		}
		shares, err := ShamirSplit(checksummedSecret(secret), n, k)
		if err != nil {
			fail(err.Error())
		}
		fmt.Printf("✓ identité %s découpée en %d parts (seuil %d-sur-%d)\n", fingerprint(id.Public()), n, k, n)
		fmt.Println("  Conserve chaque part séparément. Il en faut", k, "pour reconstituer.")
		for i, sh := range shares {
			fmt.Printf("  part %d/%d : %s\n", i+1, n, hex.EncodeToString(sh))
		}

	case "combine":
		var force bool
		var parts []string
		for _, a := range args[1:] {
			if a == "--force" {
				force = true
				continue
			}
			parts = append(parts, a)
		}
		if len(parts) < 2 {
			fail("fournis au moins le seuil de parts : communikey recovery combine <p1> <p2> … [--force]")
		}
		var shares [][]byte
		for _, p := range parts {
			b, err := hex.DecodeString(p)
			if err != nil {
				fail("part non hexadécimale: " + p)
			}
			shares = append(shares, b)
		}
		combined, err := ShamirCombine(shares)
		if err != nil {
			fail(err.Error())
		}
		// Sous le seuil K, Shamir renvoie une valeur bien formée mais FAUSSE
		// (propriété du schéma) — le checksum est ce qui distingue une vraie
		// reconstruction d'une combinaison invalide, AVANT toute dérivation.
		secret, err := verifyChecksummedSecret(combined)
		if err != nil {
			fail(err.Error())
		}
		id, err := UnmarshalIdentity(secret)
		if err != nil {
			fail("identité non reconstituée : " + err.Error())
		}
		fmt.Printf("✓ identité reconstituée : fingerprint %s\n", fingerprint(id.Public()))
		if vaultExists(s) && !force {
			fail(fmt.Sprintf("un vault existe déjà (%s) — vérifie que le fingerprint ci-dessus est bien celui attendu, puis relance avec --force pour l'écraser", identityVaultPath(s)))
		}
		if pass, ok := resolveVaultPass(); ok {
			if err := saveIdentity(s, id, pass); err != nil {
				fail(err.Error())
			}
			fmt.Printf("  vault ré-écrit : %s\n", identityVaultPath(s))
		} else {
			fmt.Println("  (définis COMKEY_VAULT_PASS_FILE ou COMKEY_VAULT_PASS pour ré-écrire le vault)")
		}

	case "phrase":
		pass, ok := resolveVaultPass()
		if !ok {
			fail("définis COMKEY_VAULT_PASS_FILE ou COMKEY_VAULT_PASS pour ouvrir le vault")
		}
		id, err := loadIdentity(s, pass)
		if err != nil {
			fail(err.Error())
		}
		master, err := id.MarshalSecret()
		if err != nil {
			fail(err.Error())
		}
		phrase, err := EntropyToMnemonic(master)
		if err != nil {
			fail(err.Error())
		}
		fmt.Printf("Phrase de récupération (24 mots) de l'identité %s.\n", fingerprint(id.Public()))
		fmt.Print("Conserve-la HORS-LIGNE (papier, coffre). Quiconque l'a possède l'identité.\n\n")
		fmt.Printf("  %s\n", phrase)

	case "from-phrase":
		// Accepte la phrase entre guillemets (1 arg) OU mot par mot (N args).
		// --force peut apparaître n'importe où (aucun mot de la wordlist BIP-39
		// ne s'écrit "--force").
		var force bool
		var words []string
		for _, a := range args[1:] {
			if a == "--force" {
				force = true
				continue
			}
			words = append(words, a)
		}
		if len(words) < 1 {
			fail("usage: communikey recovery from-phrase [--force] \"<12 à 24 mots>\"")
		}
		master, err := MnemonicToEntropy(strings.Join(words, " "))
		if err != nil {
			fail(err.Error())
		}
		id, err := UnmarshalIdentity(master)
		if err != nil {
			fail(err.Error())
		}
		fmt.Printf("✓ identité reconstituée depuis la phrase : %s\n", fingerprint(id.Public()))
		if vaultExists(s) && !force {
			fail(fmt.Sprintf("un vault existe déjà (%s) — vérifie que le fingerprint ci-dessus est bien celui attendu, puis relance avec --force pour l'écraser", identityVaultPath(s)))
		}
		if pass, ok := resolveVaultPass(); ok {
			if err := saveIdentity(s, id, pass); err != nil {
				fail(err.Error())
			}
			fmt.Printf("  vault ré-écrit : %s\n", identityVaultPath(s))
		} else {
			fmt.Println("  (définis COMKEY_VAULT_PASS_FILE ou COMKEY_VAULT_PASS pour ré-écrire le vault)")
		}

	default:
		fail("sous-commande inconnue: " + args[0] + " (split | combine | phrase | from-phrase)")
	}
}
