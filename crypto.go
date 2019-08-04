// Reference:
// https://gist.github.com/miguelmota/3ea9286bd1d3c2a985b67cac4ba2130a
// https://stackoverflow.com/questions/48958304/pkcs1-and-pkcs8-format-for-rsa-private-key
// https://www.thepolyglotdeveloper.com/2018/02/encrypt-decrypt-data-golang-application-crypto-packages/
// https://tutorialedge.net/golang/go-encrypt-decrypt-aes-tutorial/
package util

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

// GenerateKeyPair generates a new key pair
func GenerateKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	var err error
	var privKey *rsa.PrivateKey

	// TODO: find out whether it's a good idea to use rand.Reader (global variables)
	if privKey, err = rsa.GenerateKey(rand.Reader, bits); err != nil {
		return nil, nil, err
	}

	return privKey, &privKey.PublicKey, err
}

// PrivateKeyToBytes private key to bytes
func PrivateKeyToBytes(priv *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(priv),
		},
	)
}

// PublicKeyToBytes public key to bytes
func PublicKeyToBytes(pub *rsa.PublicKey) ([]byte, error) {
	var err error
	var pubASN1 []byte

	if pubASN1, err = x509.MarshalPKIXPublicKey(pub); err != nil {
		return nil, err
	}

	//
	var pubBytes []byte

	pubBytes = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubASN1,
	})

	return pubBytes, nil
}

// BytesToPrivateKey bytes to private key
func BytesToPrivateKey(data []byte) (*rsa.PrivateKey, error) {
	var err error
	block, _ := pem.Decode(data)
	b := block.Bytes

	if x509.IsEncryptedPEMBlock(block) == true {
		if b, err = x509.DecryptPEMBlock(block, nil); err != nil {
			return nil, err
		}
	}

	//
	var privKey *rsa.PrivateKey

	switch block.Type {
	case "RSA PRIVATE KEY":
		// pkcs1
		if privKey, err = x509.ParsePKCS1PrivateKey(b); err != nil {
			return nil, err
		}
	case "PRIVATE KEY":
		// pkcs8
		var ifc interface{}
		var ok bool

		if ifc, err = x509.ParsePKCS8PrivateKey(b); err != nil {
			return nil, err
		}

		if privKey, ok = ifc.(*rsa.PrivateKey); !ok {
			return nil, fmt.Errorf("Failed to type assertion to *rsa.PrivateKey")
		}
	default:
		return nil, fmt.Errorf("unsupported %s block.Type", block.Type)
	}

	return privKey, nil
}

// BytesToPublicKey bytes to public key
func BytesToPublicKey(data []byte) (*rsa.PublicKey, error) {
	var err error
	block, _ := pem.Decode(data)
	b := block.Bytes

	if x509.IsEncryptedPEMBlock(block) == true {
		if b, err = x509.DecryptPEMBlock(block, nil); err != nil {
			return nil, err
		}
	}

	//
	var ifc interface{}

	if ifc, err = x509.ParsePKIXPublicKey(b); err != nil {
		return nil, err
	}

	//
	var pubKey *rsa.PublicKey
	var ok bool

	if pubKey, ok = ifc.(*rsa.PublicKey); !ok {
		return nil, fmt.Errorf("Failed to type assert to *rsa.PublicKey")
	}

	return pubKey, nil
}

// EncryptWithPublicKey encrypts data with public key
func EncryptWithPublicKey(msg []byte, pub *rsa.PublicKey) ([]byte, error) {
	hash := sha512.New()
	return rsa.EncryptOAEP(hash, rand.Reader, pub, msg, nil)
}

// DecryptWithPrivateKey decrypts data with private key
func DecryptWithPrivateKey(ciphertext []byte, priv *rsa.PrivateKey) ([]byte, error) {
	hash := sha512.New()
	return rsa.DecryptOAEP(hash, rand.Reader, priv, ciphertext, nil)
}

// HashPassword ...
func HashPassword(plaintextPassword string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(plaintextPassword), bcrypt.DefaultCost)
}

// ValidatePassword ...
func ValidatePassword(hashed string, plaintextPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plaintextPassword))
}

// CreateHash ...
func CreateHash(data string) []byte {
	digest := sha256.Sum256([]byte(data))

	// Alternative way
	//hash := sha256.New()
	//hash.Write([]byte(data))
	//digest := hash.Sum(nil)

	return digest[:]
}

// HMACHash hashes data using a secret key
func HMACHash(message string, secret string) string {
	hash := hmac.New(sha256.New, []byte(secret))
	hash.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

// EncryptAES ...
func EncryptAES(data []byte, passphrase string) ([]byte, error) {
	// Generate a new aes cipher using our 32 byte long key
	var err error
	var block cipher.Block

	if block, err = aes.NewCipher(CreateHash(passphrase)); err != nil {
		return nil, err
	}

	// gcm or Galois/Counter Mode, is a mode of operation
	// for symmetric key cryptographic block ciphers
	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	var gcm cipher.AEAD

	if gcm, err = cipher.NewGCM(block); err != nil {
		return nil, err
	}

	// creates a new byte array the size of the nonce
	// which must be passed to Seal
	nonce := make([]byte, gcm.NonceSize())

	// populates our nonce with a cryptographically secure
	// random sequence
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// here we encrypt our text using the Seal function
	// Seal encrypts and authenticates plaintext, authenticates the
	// additional data and appends the result to dst, returning the updated
	// slice. The nonce must be NonceSize() bytes long and unique for all
	// time, for a given key.
	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	return ciphertext, nil
}

// DecryptAES ...
func DecryptAES(data []byte, passphrase string) ([]byte, error) {
	var err error
	var block cipher.Block

	key := CreateHash(passphrase)

	if block, err = aes.NewCipher(key); err != nil {
		return nil, err
	}

	//
	var gcm cipher.AEAD

	if gcm, err = cipher.NewGCM(block); err != nil {
		return nil, err
	}

	//
	nonceSize := gcm.NonceSize()

	// TODO: do we really need this checking?
	if len(data) < nonceSize {
		return nil, fmt.Errorf("data size is less than nonceSize")
	}

	//
	var plaintext []byte

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	if plaintext, err = gcm.Open(nil, nonce, ciphertext, nil); err != nil {
		return nil, err
	}

	return plaintext, nil
}

// SignSignature signs the data with a private key
func SignSignature(privateKey *rsa.PrivateKey, data []byte) ([]byte, error) {
	digest := sha256.Sum256(data)
	return rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
}

// VerifySignature verifies the data with a public key
func VerifySignature(publicKey *rsa.PublicKey, data []byte, sig []byte) error {
	digest := sha256.Sum256(data)
	return rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], sig)
}
