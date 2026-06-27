package main

// recovery.go — CLI de recovery par parts Shamir sur l'identité du vault.
//
//	csend recovery split <K> <N>     découpe le secret d'identité en N parts (K-sur-N)
//	csend recovery combine <p…>      reconstitue l'identité depuis ≥ K parts
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
)

func loadIdentity(s *Store, pass []byte) (*Identity, error) {
	data, err := os.ReadFile(identityVaultPath(s))
	if err != nil {
		return nil, fmt.Errorf("aucun vault d'identité (csend id --create d'abord): %w", err)
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
		fail("usage: csend recovery split <K> <N>  |  csend recovery combine <part…>")
	}
	s := mustStore()
	switch args[0] {
	case "split":
		if len(args) < 3 {
			fail("usage: csend recovery split <K> <N>  (K = seuil, N = nb de parts)")
		}
		k, err1 := strconv.Atoi(args[1])
		n, err2 := strconv.Atoi(args[2])
		if err1 != nil || err2 != nil {
			fail("K et N doivent être des entiers")
		}
		pass := os.Getenv("CSEND_VAULT_PASS")
		if pass == "" {
			fail("définis CSEND_VAULT_PASS pour ouvrir le vault à découper")
		}
		id, err := loadIdentity(s, []byte(pass))
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
			fail("fournis au moins le seuil de parts : csend recovery combine <p1> <p2> …")
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
		if pass := os.Getenv("CSEND_VAULT_PASS"); pass != "" {
			if err := saveIdentity(s, id, []byte(pass)); err != nil {
				fail(err.Error())
			}
			fmt.Printf("  vault ré-écrit : %s\n", identityVaultPath(s))
		} else {
			fmt.Println("  (définis CSEND_VAULT_PASS pour ré-écrire le vault local)")
		}

	default:
		fail("sous-commande inconnue: " + args[0] + " (split | combine)")
	}
}
