package genesis

import (
	"encoding/hex"
	"fmt"

	"github.com/exantech/moneroutil"
)

type GenesisBlockInfo struct {
	Hash      moneroutil.Hash
	Header    []byte
	Timestamp uint32
	TxBlob    []byte
}

func GetGenesisBlockInfo(network string) *GenesisBlockInfo {
	if network == "mainnet" {
		txHex := "013c01ff0001ffffffffffff03029b2e4c0281c0b02e7c53291a94d1d0cbff8883f8024f5142ee494ffbbd08807121017767aafcde9be00dcfd098715ebcf7f410daebc582fda69d24a28e9d0bc890d1"
		txBlob, _ := hex.DecodeString(txHex)

		return &GenesisBlockInfo{
			Hash:   moneroutil.Hash{0x41, 0x80, 0x15, 0xbb, 0x9a, 0xe9, 0x82, 0xa1, 0x97, 0x5d, 0xa7, 0xd7, 0x92, 0x77, 0xc2, 0x70, 0x57, 0x27, 0xa5, 0x68, 0x94, 0xba, 0x0f, 0xb2, 0x46, 0xad, 0xaa, 0xbb, 0x1f, 0x46, 0x32, 0xe3},
			Header: []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00},
			TxBlob: txBlob,
		}

	} else if network == "stagenet" {
		txHex := "013c01ff0001ffffffffffff0302df5d56da0c7d643ddd1ce61901c7bdc5fb1738bfe39fbe69c28a3a7032729c0f2101168d0c4ca86fb55a4cf6a36d31431be1c53a3bd7411bb24e8832410289fa6f3b"
		txBlob, _ := hex.DecodeString(txHex)

		return &GenesisBlockInfo{
			Hash:   moneroutil.Hash{0x76, 0xee, 0x3c, 0xc9, 0x86, 0x46, 0x29, 0x22, 0x06, 0xcd, 0x3e, 0x86, 0xf7, 0x4d, 0x88, 0xb4, 0xdc, 0xc1, 0xd9, 0x37, 0x08, 0x86, 0x45, 0xe9, 0xb0, 0xcb, 0xca, 0x84, 0xb7, 0xce, 0x74, 0xeb},
			Header: []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x12, 0x27, 0x00, 0x00},
			TxBlob: txBlob,
		}
	}

	panic(fmt.Sprintf("unknown network type: %s", network))
}
