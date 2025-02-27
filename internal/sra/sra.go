package sra

// SRA - RSA variant

import (
	"crypto/rand"
	"math/big"
)

// Function to generate two large prime numbers
func generateLargePrime(bits int) (*big.Int, *big.Int, error) {
	p, err := rand.Prime(rand.Reader, bits)
	if err != nil {
		return nil, nil, err
	}

	var q *big.Int
	for {
		q, err = rand.Prime(rand.Reader, bits)
		if err != nil {
			return nil, nil, err
		}
		if p.Cmp(q) != 0 { // Ensure distinct primes
			break
		}
	}

	return p, q, nil
}

// Greatest Common Divisor between 2 numbers and return it
func gcd(a, b *big.Int) *big.Int {
	return new(big.Int).GCD(nil, nil, a, b)
}

// Generates a random number and checks if it is co-prime with x
// Used when generating keys, x will always be Eulers Totient
func generateRandomCoPrime(x *big.Int) (*big.Int, error) {
	for {
		// Generate a random number in the range [2, max)
		randomNum, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), uint(2048))) // Limit size is 2048 bits
		if err != nil {
			return nil, err
		}
		randomNum.Add(randomNum, big.NewInt(2)) // Adjust to ensure it's at least 2

		// Check if GCD is 1
		if gcd(x, randomNum).Cmp(big.NewInt(1)) == 0 {
			return randomNum, nil
		}
	}
}
