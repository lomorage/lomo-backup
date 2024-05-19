package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"

	lomoio "github.com/lomorage/lomo-backup/common/io"
	"golang.org/x/crypto/argon2"
)

func SaltLen() int {
	return aes.BlockSize
}

type Encryptor struct {
	sreader *lomoio.CryptoStreamReader
}

func newCipherStream(key, iv []byte) (cipher.Stream, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return cipher.NewCTR(block, iv), nil
}

// Encrytor wrap io.CryptStreamReader and create ciper.Stream automatically
func NewEncryptor(r io.ReadSeeker, key, iv []byte, hasHeader bool) (*Encryptor, error) {
	stream, err := newCipherStream(key, iv)
	if err != nil {
		return nil, err
	}

	en := &Encryptor{}
	if hasHeader {
		en.sreader, err = lomoio.NewCryptoStreamReader(r, iv, stream)
	} else {
		en.sreader, err = lomoio.NewCryptoStreamReader(r, nil, stream)
	}
	return en, err
}

func (e *Encryptor) Read(p []byte) (int, error) {
	return e.sreader.Read(p)
}

func (e *Encryptor) Seek(offset int64, whence int) (int64, error) {
	return e.sreader.Seek(offset, whence)
}

func (e *Encryptor) GetHash() []byte {
	return e.sreader.GetHash()
}

type Decryptor struct {
	swriter *lomoio.CryptoStreamWriter
}

// Decrytor wrap io.CryptStreamWriter and create ciper.Stream automatically
func NewDecryptor(w io.Writer, key, iv []byte) (*Decryptor, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewCTR(block, iv)

	de := &Decryptor{}
	de.swriter = lomoio.NewCryptoStreamWriter(w, stream)
	return de, nil
}

func (d *Decryptor) Write(p []byte) (int, error) {
	return d.swriter.Write(p)
}

// MasterDecryptor will on-the-fly decrypt nonce
type MasterDecryptor struct {
	masterKey []byte
	writer    io.Writer
	decryptor *Decryptor
}

// The Decrytor wrap io.CryptStreamWriter and create ciper.Stream automatically
func NewMasterDecryptor(w io.Writer, key []byte) *MasterDecryptor {
	return &MasterDecryptor{masterKey: key, writer: w}
}

func (md *MasterDecryptor) Write(p []byte) (int, error) {
	if md.decryptor == nil {
		if len(p) < aes.BlockSize {
			return 0, fmt.Errorf("decrypt write buffer need %d at least, got %d", aes.BlockSize, len(p))
		}
		iv := p[:aes.BlockSize]
		var err error
		md.decryptor, err = NewDecryptor(md.writer, DeriveKeyFromMasterKey(md.masterKey, iv), iv)
		if err != nil {
			return 0, err
		}
		n, err := md.decryptor.Write(p[aes.BlockSize:])
		return n + aes.BlockSize, err
	}
	return md.decryptor.Write(p)
}

const (
	argon2Time      = 1
	argon2Memory    = 64 * 1024
	argon2Thread    = 4
	argon2KeyLength = 32 // 32 bytes key for AES-256
)

// DeriveKeyFromPassphrase derives a cryptographic key from the provided passphrase using Argon2.
func DeriveKeyFromMasterKey(masterKey, salt []byte) []byte {
	// Use Argon2id for key derivation
	return argon2.IDKey(masterKey, salt, argon2Time, argon2Memory, argon2Thread, argon2KeyLength)
}
