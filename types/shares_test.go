package types

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/celestiaorg/nmt/namespace"
	"github.com/tendermint/tendermint/internal/libs/protoio"
	"github.com/tendermint/tendermint/pkg/consts"
)

type splitter interface {
	splitIntoShares() NamespacedShares
}

func TestMakeShares(t *testing.T) {
	reservedTxNamespaceID := append(bytes.Repeat([]byte{0}, 7), 1)
	reservedEvidenceNamespaceID := append(bytes.Repeat([]byte{0}, 7), 3)
	val := NewMockPV()
	blockID := makeBlockID([]byte("blockhash"), 1000, []byte("partshash"))
	blockID2 := makeBlockID([]byte("blockhash2"), 1000, []byte("partshash"))
	vote1 := makeVote(t, val, "chainID", 0, 10, 2, 1, blockID, defaultVoteTime)
	vote2 := makeVote(t, val, "chainID", 0, 10, 2, 1, blockID2, defaultVoteTime)
	testEvidence := &DuplicateVoteEvidence{
		VoteA: vote1,
		VoteB: vote2,
	}
	testEvidenceBytes, err := protoio.MarshalDelimited(testEvidence.ToProto())
	largeTx := Tx(bytes.Repeat([]byte("large Tx"), 50))
	largeTxLenDelimited, _ := largeTx.MarshalDelimited()
	smolTx := Tx("small Tx")
	smolTxLenDelimited, _ := smolTx.MarshalDelimited()
	msg1 := Message{
		NamespaceID: namespace.ID("8bytesss"),
		Data:        []byte("some data"),
	}
	msg1Marshaled, _ := msg1.MarshalDelimited()
	if err != nil {
		t.Fatalf("Could not encode evidence: %v, error: %v\n", testEvidence, err)
	}

	type args struct {
		data splitter
	}
	tests := []struct {
		name string
		args args
		want NamespacedShares
	}{
		{"evidence",
			args{
				data: &EvidenceData{
					Evidence: []Evidence{testEvidence},
				},
			}, NamespacedShares{NamespacedShare{
				Share: append(
					append(reservedEvidenceNamespaceID, byte(0)),
					testEvidenceBytes[:consts.TxShareSize]...,
				),
				ID: reservedEvidenceNamespaceID,
			}, NamespacedShare{
				Share: append(
					append(reservedEvidenceNamespaceID, byte(0)),
					zeroPadIfNecessary(testEvidenceBytes[consts.TxShareSize:], consts.TxShareSize)...,
				),
				ID: reservedEvidenceNamespaceID,
			}},
		},
		{"small LL Tx",
			args{
				data: Txs{smolTx},
			},
			NamespacedShares{
				NamespacedShare{
					Share: append(
						append(reservedTxNamespaceID, byte(0)),
						zeroPadIfNecessary(smolTxLenDelimited, consts.TxShareSize)...,
					),
					ID: reservedTxNamespaceID,
				},
			},
		},
		{"one large LL Tx",
			args{
				data: Txs{largeTx},
			},
			NamespacedShares{
				NamespacedShare{
					Share: append(
						append(reservedTxNamespaceID, byte(0)),
						largeTxLenDelimited[:consts.TxShareSize]...,
					),
					ID: reservedTxNamespaceID,
				},
				NamespacedShare{
					Share: append(
						append(reservedTxNamespaceID, byte(0)),
						zeroPadIfNecessary(largeTxLenDelimited[consts.TxShareSize:], consts.TxShareSize)...,
					),
					ID: reservedTxNamespaceID,
				},
			},
		},
		{"large then small LL Tx",
			args{
				data: Txs{largeTx, smolTx},
			},
			NamespacedShares{
				NamespacedShare{
					Share: append(
						append(reservedTxNamespaceID, byte(0)),
						largeTxLenDelimited[:consts.TxShareSize]...,
					),
					ID: reservedTxNamespaceID,
				},
				NamespacedShare{
					Share: append(
						append(reservedTxNamespaceID, byte(len(largeTxLenDelimited)-consts.TxShareSize+consts.NamespaceSize+consts.ShareReservedBytes)),
						zeroPadIfNecessary(
							append(largeTxLenDelimited[consts.TxShareSize:], smolTxLenDelimited...),
							consts.TxShareSize,
						)...,
					),
					ID: reservedTxNamespaceID,
				},
			},
		},
		{"ll-app message",
			args{
				data: Messages{[]Message{msg1}},
			},
			NamespacedShares{
				NamespacedShare{
					Share: append(
						[]byte(msg1.NamespaceID),
						zeroPadIfNecessary(msg1Marshaled, consts.MsgShareSize)...,
					),
					ID: msg1.NamespaceID,
				},
			},
		},
	}
	for i, tt := range tests {
		tt := tt // stupid scopelint :-/
		i := i
		t.Run(tt.name, func(t *testing.T) {
			got := tt.args.data.splitIntoShares()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%v: makeShares() = \n%+v\nwant\n%+v\n", i, got, tt.want)
			}
		})
	}
}

func Test_zeroPadIfNecessary(t *testing.T) {
	type args struct {
		share []byte
		width int
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{"pad", args{[]byte{1, 2, 3}, 6}, []byte{1, 2, 3, 0, 0, 0}},
		{"not necessary (equal to shareSize)", args{[]byte{1, 2, 3}, 3}, []byte{1, 2, 3}},
		{"not necessary (greater shareSize)", args{[]byte{1, 2, 3}, 2}, []byte{1, 2, 3}},
	}
	for _, tt := range tests {
		tt := tt // stupid scopelint :-/
		t.Run(tt.name, func(t *testing.T) {
			if got := zeroPadIfNecessary(tt.args.share, tt.args.width); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("zeroPadIfNecessary() = %v, want %v", got, tt.want)
			}
		})
	}
}
