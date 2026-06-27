package main

// shamir.go — recovery par SEUIL : Shamir Secret Sharing sur GF(2^8).
//
// La « math complexe » demandée, faite correctement : un secret est découpé en N
// parts ; il en faut un seuil K (K-sur-N) pour le reconstituer ; K-1 parts ne
// révèlent RIEN (sûreté de seuil, information-théorique). Chaque octet du secret
// est le terme constant d'un polynôme aléatoire de degré K-1 sur GF(256), évalué
// en x=1..N ; la reconstruction = interpolation de Lagrange en x=0.
//
// Implémentation from-scratch d'un schéma STANDARD (corps GF(256)/0x11b, comme
// AES) : testée (roundtrip + propriété de seuil). Audit externe recommandé avant
// usage critique (§38/§29).

import (
	"crypto/rand"
	"errors"
	"fmt"
)

// gfMul multiplies two elements of GF(2^8) modulo the AES polynomial x^8+x^4+x^3+x+1
// (0x11b), via the standard carry-less / xtime reduction. Constant number of steps.
func gfMul(a, b byte) byte {
	var p byte
	for i := 0; i < 8; i++ {
		if b&1 != 0 {
			p ^= a
		}
		hi := a & 0x80
		a <<= 1
		if hi != 0 {
			a ^= 0x1b // implicit 0x100 bit folds back as 0x1b
		}
		b >>= 1
	}
	return p
}

// gfInv returns the multiplicative inverse via Fermat: a^(254) = a^-1 in GF(2^8).
func gfInv(a byte) byte {
	result := byte(1)
	base := a
	for e := 254; e > 0; e >>= 1 {
		if e&1 == 1 {
			result = gfMul(result, base)
		}
		base = gfMul(base, base)
	}
	return result
}

// gfEval evaluates a polynomial (coeffs[0] = constant term) at x, by Horner.
func gfEval(coeffs []byte, x byte) byte {
	var out byte
	for i := len(coeffs) - 1; i >= 0; i-- {
		out = gfMul(out, x) ^ coeffs[i]
	}
	return out
}

// gfInterpolateAtZero recovers the constant term from samples (xs[i], ys[i]) via
// Lagrange interpolation evaluated at x=0. In GF(2^8) subtraction is XOR.
func gfInterpolateAtZero(xs, ys []byte) byte {
	var result byte
	for i := range xs {
		num, den := byte(1), byte(1)
		for m := range xs {
			if m == i {
				continue
			}
			num = gfMul(num, xs[m])       // (0 - x_m) == x_m
			den = gfMul(den, xs[i]^xs[m]) // (x_i - x_m) == x_i ^ x_m
		}
		result ^= gfMul(ys[i], gfMul(num, gfInv(den)))
	}
	return result
}

// ShamirSplit splits secret into `parts` shares, any `threshold` of which recover
// it. Each share is len(secret)+1 bytes: the last byte is its x-coordinate.
func ShamirSplit(secret []byte, parts, threshold int) ([][]byte, error) {
	if threshold < 2 || threshold > parts {
		return nil, fmt.Errorf("seuil invalide: 2 ≤ K(%d) ≤ N(%d)", threshold, parts)
	}
	if parts > 255 {
		return nil, errors.New("au plus 255 parts (taille de GF(256))")
	}
	if len(secret) == 0 {
		return nil, errors.New("secret vide")
	}
	shares := make([][]byte, parts)
	for i := range shares {
		shares[i] = make([]byte, len(secret)+1)
		shares[i][len(secret)] = byte(i + 1) // x ∈ [1..parts], never 0
	}
	coeffs := make([]byte, threshold)
	for j, b := range secret {
		// Random polynomial: constant term = secret byte, rest random.
		coeffs[0] = b
		if _, err := rand.Read(coeffs[1:]); err != nil {
			return nil, err
		}
		for i := range shares {
			shares[i][j] = gfEval(coeffs, byte(i+1))
		}
	}
	return shares, nil
}

// ShamirCombine reconstructs the secret from a set of shares (need ≥ threshold,
// distinct x-coordinates). Fewer than threshold yields a wrong value, not the
// secret (that is the security property), so callers must track K themselves.
func ShamirCombine(shares [][]byte) ([]byte, error) {
	if len(shares) < 2 {
		return nil, errors.New("au moins 2 parts requises")
	}
	L := len(shares[0])
	if L < 2 {
		return nil, errors.New("part malformée")
	}
	xs := make([]byte, len(shares))
	seen := map[byte]bool{}
	for i, s := range shares {
		if len(s) != L {
			return nil, errors.New("parts de longueurs différentes")
		}
		x := s[L-1]
		if x == 0 {
			return nil, errors.New("coordonnée x nulle (part invalide)")
		}
		if seen[x] {
			return nil, errors.New("coordonnées x dupliquées")
		}
		seen[x] = true
		xs[i] = x
	}
	secret := make([]byte, L-1)
	ys := make([]byte, len(shares))
	for j := 0; j < L-1; j++ {
		for i, s := range shares {
			ys[i] = s[j]
		}
		secret[j] = gfInterpolateAtZero(xs, ys)
	}
	return secret, nil
}
