package sra

import (
	"fmt"
	"math/big"
	"testing"
)

func TestGenerateLargePrime(t *testing.T) {
	tests := []struct {
		name string
		bits int
	}{
		{"64-bit primes", 64},
		{"128-bit primes", 128},
		{"256-bit primes", 256},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, q, err := generateLargePrime(tt.bits)
			if err != nil {
				t.Fatalf("Error generating primes: %v", err)
			}

			// Verify primality
			if !p.ProbablyPrime(20) || !q.ProbablyPrime(20) {
				t.Error("Generated numbers are not prime")
			}

			// Verify bit length
			if p.BitLen() != tt.bits || q.BitLen() != tt.bits {
				t.Errorf("Incorrect bit length. Expected %d, got p:%d q:%d",
					tt.bits, p.BitLen(), q.BitLen())
			}

			// Verify distinct primes
			if p.Cmp(q) == 0 {
				t.Error("Generated identical primes")
			}
		})
	}
}

func TestGCD(t *testing.T) {
	tests := []struct {
		a        int64
		b        int64
		expected int64
	}{
		{6, 9, 3},
		{35, 14, 7},
		{17, 23, 1},
		{0, 5, 5},
		{0, 0, 0},
		{42, 42, 42},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("gcd(%d,%d)", tt.a, tt.b), func(t *testing.T) {
			a := big.NewInt(tt.a)
			b := big.NewInt(tt.b)
			result := gcd(a, b)

			if result.Int64() != tt.expected {
				t.Errorf("Expected GCD %d, got %d", tt.expected, result.Int64())
			}
		})
	}
}

func TestGenerateRandomCoPrime(t *testing.T) {
	testCases := []struct {
		name string
		x    *big.Int
	}{
		{"small even", big.NewInt(10)},
		{"prime", big.NewInt(17)},
		{"composite", big.NewInt(100)},
		{"power of two", big.NewInt(32)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for i := 0; i < 5; i++ { // Multiple trials to test randomness
				coPrime, err := generateRandomCoPrime(tc.x)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				// Verify coprimality
				g := gcd(coPrime, tc.x)
				if g.Cmp(big.NewInt(1)) != 0 {
					t.Errorf("Numbers are not coprime. GCD: %s", g)
				}

				// Verify value range
				if coPrime.Cmp(big.NewInt(2)) < 0 {
					t.Error("Generated number is smaller than 2")
				}

				max := new(big.Int).Lsh(big.NewInt(1), 2048)
				if coPrime.Cmp(max) >= 0 {
					t.Error("Generated number exceeds 2^2048")
				}

				// Special case checks
				if tc.x.Bit(0) == 0 { // If x is even
					if coPrime.Bit(0) == 0 {
						t.Error("Generated even number for even x")
					}
				}
			}
		})
	}
}
