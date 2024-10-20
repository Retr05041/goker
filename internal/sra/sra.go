package crypto

// SRA - RSA variant

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// Function to generate two large prime numbers
func generateLargePrime(bits int) (*big.Int, error) {
	return rand.Prime(rand.Reader, bits)
}

// Greatest Common Divisor between 2 numbers
func gcd(a, b *big.Int) *big.Int {
	zero := big.NewInt(0) // For efficiency - not declaring it each iteration
	oldB := new(big.Int)
	for b.Cmp(zero) != 0 {
		oldB.Set(b)
		b.Set(new(big.Int).Mod(a, b))
		a.Set(oldB)
	}
	return a
}

// Modular inverse of a modulo m - using Extended Euclidean Algorithm
// return x, such that `(a*x) mod m = 1`
func modInverse(a, m *big.Int) (*big.Int, error) {
	m0 := new(big.Int).Set(m) // store m for later use
	// These will represent the coefficients of the linear combination
	y := big.NewInt(0)
	x := big.NewInt(1)

	// every number is congruent to 0 modulo 1
	if m.Cmp(big.NewInt(1)) == 0 {
		return nil, fmt.Errorf("modular inverse does not exist")
	}

	for a.Cmp(big.NewInt(1)) > 0 {
		quotent := new(big.Int).Div(a, m)
		t := new(big.Int).Set(m) // store m for this iteration

		m.Set(new(big.Int).Mod(a, m)) // m = a mod m
		a = t                                        // set previous value of m becomes a
		t = new(big.Int).Set(y)                      // set the hold var to old y

		y = new(big.Int).Set(new(big.Int).Sub(x, new(big.Int).Mul(quotent, y))) // y = x-(quotent * m)
		x = t                                                                   // x now becomes old y
	}

	// if x is negative, adjust by adding m0 to ensure the result is positive
	if x.Cmp(big.NewInt(0)) < 0 {
		x = new(big.Int).Add(x, m0)
	}

	return x, nil
}

// Key generation - returns: private key, public key, modulus
func GenerateKeys() (*big.Int, *big.Int, *big.Int, error) {
	// generate p and q
	p, err := generateLargePrime(8)
	if err != nil {
		return nil, nil, nil, err
	}
	q, err := generateLargePrime(8)
	if err != nil {
		return nil, nil, nil, err
	}

	// n = p * q
	n := new(big.Int).Mul(p, q)

	// Eulers Totient -- ϕ(n) = (p−1)(q−1)
	// Used for caluclating private keys
	phi := new(big.Int).Mul(new(big.Int).Sub(p, big.NewInt(1)), new(big.Int).Sub(q, big.NewInt(1)))

	// Choose e1 such that gcd(e1, phi(n)) = 1
	//e1 := big.NewInt(3)
	//for gcd(e1, phi).Cmp(big.NewInt(1)) != 0 {
	//	e1 = new(big.Int).Set(e1.Add(e1, big.NewInt(2))) // Ensure e1 is odd
	//}

	// Choose e such that gcd(e, phi(n)) = 1
	e1 := big.NewInt(65537) // Common choice for e
	if gcd(e1, phi).Cmp(big.NewInt(1)) != 0 {
		return nil, nil, nil, fmt.Errorf("e is not coprime with phi")
	}

	d1, err := modInverse(e1, phi)
	if err != nil {
		return nil, nil, nil, err
	}

	return e1, d1, n, nil
}

// Converts a string to a base-256 big int for encryption
func stringToBigInt(str string) *big.Int {
	result := big.NewInt(0)

	bytes := []byte(str)

	for _, byte := range bytes {
		result.Lsh(result, 8)                       // // Shift left by 8 bits (multiply by 256)
		result.Add(result, big.NewInt(int64(byte))) // Add the bytes value
	}

	return result
}

// Convert a base-256 big int to string for decrption
func bigIntToString(num *big.Int) string {
	if num.Sign() == 0 {
		return ""
	}

	var result []byte

	// Create a big.Int for 256
	base := big.NewInt(256)
	remainder := big.NewInt(0)

	for num.Cmp(big.NewInt(0)) > 0 {
		// Divide num by 256 and get the remainder
		remainder.Set(num.Mod(num, base))                // m = num % 256
		result = append(result, byte(remainder.Int64())) // Append character to result
		num.Div(num, base)                               // n = num / 256
	}

    // Reverse the byte slice before converting to string
    for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
        result[i], result[j] = result[j], result[i]
    }

    return string(result) // Convert byte slice to string
}

// Encryption
func Encrypt(message string, publicKey, modN *big.Int) *big.Int {
	numMsg := stringToBigInt(message)
	fmt.Println("Numberfied message: " + numMsg.String())
	return new(big.Int).Exp(numMsg, publicKey, modN) // ciphertext = numMsg^publicKey `mod` modN
}

// Decryption
func Decrypt(cipherText, privateKey, modN *big.Int) string {
	numMsg := new(big.Int).Exp(cipherText, privateKey, modN) // plaintext = ciphertext^privateKey `mod` modN
	fmt.Println("Decrypted numberfied message: " + numMsg.String())
	return bigIntToString(numMsg)
}
