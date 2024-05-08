package hash

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testFilename = "../testdata/indepedant_declaration.txt"

// hex hash is calculated by "sha256sum indepedant_declaration.txt"
const expectHexHash = "4cfd75f4c8aa09ff1992e493999fd1c0333da4393fffd9d207b60c6dc2516263"

// base64 hash is calculated by "cat indepedant_declaration.txt | openssl dgst -binary -sha256 | openssl base64 -A"
const expectBase64Hash = "TP119MiqCf8ZkuSTmZ/RwDM9pDk//9nSB7YMbcJRYmM="

// split file using command "split --bytes=1000 --numeric-suffixes=1 --suffix-length=1 ./indepedant_declaration.txt test"
// for i in `seq 1 9`; do sha256sum test$i; done | awk -F' ' '{print $1}'
var expectHexHashMultiparts = []string{
	"966b6a905a846ca688fd1f9e08882c43ec52bff8852f69139d5c720b7361d30f",
	"18e05d53abcd5c1267a762347bb6bf0ac5948524932cd4c544914372832547e1",
	"648d21f3ba5b237581f691d68e2f596dd36bfc687bbf41cca4e2c8f744e5e309",
	"fd3f7646b387a922b7a9912200c703381dee8f41ba2fb56f6d9f1e4c6c9304eb",
	"19183c6d710f85e3c0d36e5c1fbf0e1fb42b832a1471a745c11aa8d9fed85f79",
	"5d350af5e1f3e149e1f6cefe595e18a71951b306f00ef304177b7999745f4c3b",
	"42664de22a93b15140c8e882f7d5deb9a5fb29e937e6c0bef81fedea50a250f9",
	"a5747c2ec4cedc33709a361318ddf658a5afa7a12c889e7315bf0adaa82c0086",
	"eb05dc46b7f61ed556a96ee44fd68e5ebe8012926901c56c5578b7f363134360",
}

// for i in `seq 1 9`; do cat test$i | openssl dgst -binary -sha256 | openssl base64 -A; echo;done
var expectBase64HashMultiparts = []string{
	"lmtqkFqEbKaI/R+eCIgsQ+xSv/iFL2kTnVxyC3Nh0w8=",
	"GOBdU6vNXBJnp2I0e7a/CsWUhSSTLNTFRJFDcoMlR+E=",
	"ZI0h87pbI3WB9pHWji9ZbdNr/Gh7v0HMpOLI90Tl4wk=",
	"/T92RrOHqSK3qZEiAMcDOB3uj0G6L7VvbZ8eTGyTBOs=",
	"GRg8bXEPhePA025cH78OH7QrgyoUcadFwRqo2f7YX3k=",
	"XTUK9eHz4Unh9s7+WV4YpxlRswbwDvMEF3t5mXRfTDs=",
	"QmZN4iqTsVFAyOiC99XeuaX7Kek35sC++B/t6lCiUPk=",
	"pXR8LsTO3DNwmjYTGN32WKWvp6EsiJ5zFb8K2qgsAIY=",
	"6wXcRrf2HtVWqW7kT9aOXr6AEpJpAcVsVXi382MTQ2A=",
}

func TestCalculateHash(t *testing.T) {
	assert := assert.New(t)

	hash, err := CalculateHash(testFilename)

	// assert for nil (good for errors)
	assert.Nil(err)

	// assert equality
	// calculated by command "sha256sum indepedant_declaration.txt"
	assert.Equal(expectHexHash, CalculateHashHex(hash), "they should be equal")

	// calculated by command "cat indepedant_declaration.txt | openssl dgst -binary -sha256 | openssl base64 -A"
	assert.Equal(expectBase64Hash, CalculateHashBase64(hash), "they should be equal")
}

func TestCalculateMultiPartsHash(t *testing.T) {
	assert := assert.New(t)

	partsHash, err := CalculateMultiPartsHash(testFilename, 1000)

	// assert for nil (good for errors)
	assert.Nil(err)

	// assert equality
	hexParts := make([]string, len(partsHash))
	base64Parts := make([]string, len(partsHash))
	for i, p := range partsHash {
		hexParts[i] = CalculateHashHex(p)
		base64Parts[i] = CalculateHashBase64(p)
	}
	require.Equal(t, expectHexHashMultiparts, hexParts)

	require.Equal(t, expectBase64HashMultiparts, base64Parts)
}

func TestConcatAndCalculateBase64Hash(t *testing.T) {
	parts := [][]byte{}
	for _, p := range []string{
		"lzeb6gPr4raiM1LG0ZNF2OOtdoUCRu+6ewNA0Qir4sI=",
		"lZh9FyuGrF/0Vbw8CBtSFVMX04SgRnLbrPX9BYpQRNg=",
	} {
		b, err := base64.StdEncoding.DecodeString(p)
		require.Nil(t, err)
		parts = append(parts, b)
	}

	hash, err := ConcatAndCalculateBase64Hash(parts)
	require.Nil(t, err)
	require.Equal(t, "NnO4DPqD+RLUyOycER1BKbzMv6+APV72KGFvLBNay8c=", hash)
}
