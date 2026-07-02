package main

// recovery.go — CLI de recovery par parts Shamir sur l'identité du vault.
//
//	communikey recovery split <K> <N>     découpe le secret d'identité en N parts (K-sur-N)
//	communikey recovery combine <p…>      reconstitue l'identité depuis ≥ K parts
//
// Le secret découpé est le blob d'identité (sign+x25519+mlkem). Répartir les parts
// (téléphone, YubiKey, proche, papier coffre…) : perte d'un device ≠ perte du vault,
// et aucune part isolée ne révèle rien.

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

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
		fail("usage: communikey recovery split <K> <N>  |  communikey recovery combine <part…>")
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
		shares, err := ShamirSplit(secret, n, k)
		if err != nil {
			fail(err.Error())
		}
		fmt.Printf("✓ identité %s découpée en %d parts (seuil %d-sur-%d)\n", fingerprint(id.Public()), n, k, n)
		fmt.Println("  Conserve chaque part séparément. Il en faut", k, "pour reconstituer.")
		for i, sh := range shares {
			fmt.Printf("  part %d/%d : %s\n", i+1, n, hex.EncodeToString(sh))
		}

	case "combine":
		parts := args[1:]
		if len(parts) < 2 {
			fail("fournis au moins le seuil de parts : communikey recovery combine <p1> <p2> …")
		}
		var shares [][]byte
		for _, p := range parts {
			b, err := hex.DecodeString(p)
			if err != nil {
				fail("part non hexadécimale: " + p)
			}
			shares = append(shares, b)
		}
		secret, err := ShamirCombine(shares)
		if err != nil {
			fail(err.Error())
		}
		id, err := UnmarshalIdentity(secret)
		if err != nil {
			fail("parts insuffisantes ou invalides — identité non reconstituée (" + err.Error() + ")")
		}
		fmt.Printf("✓ identité reconstituée : fingerprint %s\n", fingerprint(id.Public()))
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
		if len(args) < 2 {
			fail("usage: communikey recovery from-phrase \"<12 à 24 mots>\"")
		}
		master, err := MnemonicToEntropy(strings.Join(args[1:], " "))
		if err != nil {
			fail(err.Error())
		}
		id, err := UnmarshalIdentity(master)
		if err != nil {
			fail(err.Error())
		}
		fmt.Printf("✓ identité reconstituée depuis la phrase : %s\n", fingerprint(id.Public()))
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
