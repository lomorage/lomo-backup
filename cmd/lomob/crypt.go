package main

import (
	"crypto/aes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"syscall"

	"github.com/lomorage/lomo-backup/common/crypto"
	"github.com/lomorage/lomo-backup/common/datasize"
	lomohash "github.com/lomorage/lomo-backup/common/hash"
	lomoio "github.com/lomorage/lomo-backup/common/io"
	"github.com/urfave/cli"
	"golang.org/x/term"
)

func getMasterKey() (string, error) {
	fmt.Print("Enter Master Key: ")
	bytePassword1, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return "", err
	}

	fmt.Print("\nEnter Master Key Again: ")
	bytePassword2, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return "", err
	}
	fmt.Println()
	if string(bytePassword1) != string(bytePassword2) {
		return "", fmt.Errorf("got two different keys")
	}
	return string(bytePassword1), nil
}

func genSalt(filename string) ([]byte, error) {
	if filename == "" {
		salt := make([]byte, crypto.SaltLen())
		_, err := io.ReadFull(rand.Reader, salt)
		return salt, err
	}

	h, err := lomohash.CalculateHashFile(filename)
	if err != nil {
		return nil, err
	}
	if len(h) < crypto.SaltLen() {
		return nil, fmt.Errorf("invalid hash len '%d', less than '%d'", len(h), crypto.SaltLen())
	}
	return h[:crypto.SaltLen()], nil
}

func encryptCmd(ctx *cli.Context) error {
	var ifilename, ofilename string
	switch len(ctx.Args()) {
	case 1:
		ifilename = ctx.Args()[0]
		ofilename = ifilename + ".enc"
	case 2:
		ifilename = ctx.Args()[0]
		ofilename = ctx.Args()[1]
	default:
		return errors.New("usage: [input filename] [[output filename]]. If output filename is not given, it will be <intput filename>.enc")
	}

	var err error
	masterKey := ctx.String("encrypt-key")
	if masterKey == "" {
		masterKey, err = getMasterKey()
		if err != nil {
			return err
		}
	}

	salt, err := genSalt(ifilename)
	if err != nil {
		return err
	}

	src, err := os.Open(ifilename)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(ofilename)
	if err != nil {
		return err
	}
	defer dst.Close()

	fmt.Printf("Start encrypt '%s', and save output to '%s'\n", ifilename, ofilename)

	ps := ctx.String("part-size")
	if ps == "" {
		_, err = encryptLocalFile(src, dst, []byte(masterKey), salt, true)
		if err != nil {
			return err
		}

		fmt.Println("Finish encryption!")

		return nil
	}

	// Derive key from passphrase using Argon2
	// TODO: Using IV as salt for simplicity, change to different salt?
	encryptKey := crypto.DeriveKeyFromMasterKey([]byte(masterKey), salt)

	partSize, err := datasize.ParseString(ps)
	if err != nil {
		return err
	}

	stat, err := src.Stat()
	if err != nil {
		return err
	}

	index := 1
	remaining := stat.Size()
	var (
		start, end, curr, partLength int64
		encryptor                    *crypto.Encryptor
		prs                          *lomoio.FilePartReadSeeker
	)
	for curr = 0; remaining != 0; curr += partLength {
		if remaining < int64(partSize) {
			partLength = remaining
		} else {
			partLength = int64(partSize)
		}

		if curr == 0 {
			end = int64(int(partLength) - crypto.SaltLen())
		} else {
			start = end
			end += partLength
		}

		// create a local tmpfile and save intermittent part
		pf, err := os.Create(ofilename + ".part" + strconv.Itoa(index))
		if err != nil {
			return err
		}
		defer pf.Close()

		mw := io.MultiWriter(dst, pf)

		if prs == nil {
			prs = lomoio.NewFilePartReadSeeker(src, start, end)
		} else {
			prs.SetStartEnd(start, end)
		}

		if encryptor == nil {
			encryptor, err = crypto.NewEncryptor(prs, encryptKey, salt, false)
			if err != nil {
				return err
			}
			n, err := mw.Write(salt)
			if err != nil {
				return err
			}
			if n != len(salt) {
				return fmt.Errorf("write %d byte salt while expecting %d", n, len(salt))
			}
		}

		n, err := io.Copy(mw, encryptor)
		if err != nil {
			return err
		}

		if n != end-start {
			return fmt.Errorf("write %d byte salt while expecting %d btw [%d, %d]", n, end-start, start, end)
		}

		fmt.Printf("Created '%s'\n", pf.Name())

		index++
		remaining -= end - start
	}

	fmt.Println("Finish encryption!")

	return nil
}

func encryptLocalFile(src io.ReadSeeker, dst io.Writer, masterKey, iv []byte, hasHeader bool) ([]byte, error) {
	// Derive key from passphrase using Argon2
	// TODO: Using IV as salt for simplicity, change to different salt?
	encryptKey := crypto.DeriveKeyFromMasterKey(masterKey, iv)
	encryptor, err := crypto.NewEncryptor(src, encryptKey, iv, hasHeader)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(dst, encryptor)
	if err != nil {
		return nil, err
	}
	return encryptor.GetHashEncrypt(), nil
}

func decryptLocalFile(ctx *cli.Context) error {
	if len(ctx.Args()) != 1 {
		return errors.New("usage: [encrypted file name]")
	}

	src, err := os.Open(ctx.Args()[0])
	if err != nil {
		return err
	}
	defer src.Close()

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(src, iv); err != nil {
		return err
	}

	masterKey := ctx.String("encrypt-key")
	if masterKey == "" {
		masterKey, err = getMasterKey()
		if err != nil {
			return err
		}
	}

	var dst io.Writer
	if ctx.String("output") == "" {
		dst = os.Stdout
	} else {
		f, err := os.Create(ctx.String("output"))
		if err != nil {
			return err
		}
		defer f.Close()
		dst = f
	}

	encryptKey := crypto.DeriveKeyFromMasterKey([]byte(masterKey), iv)

	decryptor, err := crypto.NewDecryptor(dst, encryptKey, iv)
	if err != nil {
		return err
	}

	_, err = io.Copy(decryptor, src)
	if err != nil {
		return err
	}

	fmt.Println("Finish decryption!")
	return nil
}

func checkHeader(ctx *cli.Context) error {
	if len(ctx.Args()) != 2 {
		return errors.New("usage: [original file name] [encrypted file name]")
	}

	h, err := genSalt(ctx.Args()[0])
	if err != nil {
		return err
	}

	f, err := os.Open(ctx.Args()[1])
	if err != nil {
		return err
	}
	defer f.Close()

	salt := make([]byte, crypto.SaltLen())
	s, err := f.Read(salt)
	if err != nil {
		return err
	}
	if s != len(salt) {
		return fmt.Errorf("expect read %d, actual read %d", len(salt), s)
	}
	if !reflect.DeepEqual(salt, h[:crypto.SaltLen()]) {
		return errors.New("different salt detected")
	}
	fmt.Println("Salt is same as expected!")
	return nil
}
