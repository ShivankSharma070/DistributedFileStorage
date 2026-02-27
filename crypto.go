package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"io"
)

// generateID reads random bytes from rand.Reader into a buffer of 32 bytes
// and returns the hexadecimal string representation
func generateID() string {
	buf := make([]byte, 32)
	io.ReadFull(rand.Reader, buf)
	return hex.EncodeToString(buf)
}

// hashKey generates an MD5 hash of the given key and returns
// its hexadecimal string representation
func hashKey(key string) string {
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

// newEncryptionKey generates a random 32 byte key.
func newEncryptionKey() []byte {
	keyBuf := make([]byte, 32)
	io.ReadFull(rand.Reader, keyBuf)
	return keyBuf
}

// copyDecrypt reads encrypted data from io.Reader and writes decrypted
// data into a io.Writer. Decryption is done using the provided key and IV
// stored in the initial bytes of the encrypted data
func copyDecrypt(key []byte, src io.Reader, dest io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize())
	// Read iv from the initial bytes of the src
	if _, err := src.Read(iv); err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)
	return copyStream(stream, block.BlockSize(), src, dest)
}

// copyEncrypt reads data from io.Reader and writes encrypted
// data into a io.Writer. Encryption is done using the provided key
// and a randomly generated IV
func copyEncrypt(key []byte, src io.Reader, dest io.Writer) (int, error) {
	// Create a new AES cipher block using the provided key
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	// Create an initialization vector (IV) of the same size as the block
	iv := make([]byte, block.BlockSize())

	// Fill iv with a random value
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return 0, err
	}

	// Write iv to destination so that it stays at the beginning of the
	// destination and can be used by decryption
	if _, err := dest.Write(iv); err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)
	return copyStream(stream, block.BlockSize(), src, dest)
}

// copyStream is a helper function that reads data from src, applies the
// provided cipher stream to encrypt or decrypt it, and writes the result
// to dest. It reads in chunks of 32KB and returns the total bytes written
func copyStream(stream cipher.Stream, blockSize int, src io.Reader, dest io.Writer) (int, error) {
	var (
		buf = make([]byte, 32*1024)
		nw  = blockSize
	)
	for {
		n, err := src.Read(buf)

		if n > 0 {
			stream.XORKeyStream(buf, buf[:n])
			nn, err := dest.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			nw += nn
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return 0, nil
		}

	}

	return nw, nil
}
