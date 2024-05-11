package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"io"

	lomoio "github.com/lomorage/lomo-backup/common/io"
)

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
func NewEncryptor(r io.ReadSeeker, key, iv []byte) (*Encryptor, error) {
	stream, err := newCipherStream(key, iv)
	if err != nil {
		return nil, err
	}

	en := &Encryptor{}
	en.sreader, err = lomoio.NewCryptoStreamReader(r, iv, stream)
	return en, err
}

func (e *Encryptor) Read(p []byte) (int, error) {
	return e.sreader.Read(p)
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
