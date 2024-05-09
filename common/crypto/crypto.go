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

// Encrytor wrap io.CryptStreamReader and create ciper.Stream automatically
func NewEncryptor(r io.Reader, key, iv []byte) (*Encryptor, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewCTR(block, iv)

	en := &Encryptor{}
	en.sreader = lomoio.NewCryptoStreamReader(r, iv, stream)
	return en, nil
}

func (e *Encryptor) Read(p []byte) (int, error) {
	return e.sreader.Read(p)
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
