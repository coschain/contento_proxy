package utils

import (
	"encoding/binary"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"proxy/rpc"
)

func GenerateSignedTx(privateKey string,client *rpc.Client, ops ...interface{}) (*prototype.SignedTransaction, error) {
	privKey := &prototype.PrivateKeyType{}
	pk, err := prototype.PrivateKeyFromWIF(privateKey)
	if err != nil {
		return nil, err
	}
	privKey = pk

	req := &grpcpb.NonParamsRequest{}
	resp, err := client.GetStatisticsInfo(req)
	if err != nil {
		client.SetAlive(false)
		return nil, err
	}
	refBlockPrefix := binary.BigEndian.Uint32(resp.State.Dgpo.HeadBlockId.Hash[8:12])
	// occupant implement
	refBlockNum := uint32(resp.State.Dgpo.HeadBlockNumber & 0x7ff)
	tx := &prototype.Transaction{RefBlockNum: refBlockNum, RefBlockPrefix: refBlockPrefix, Expiration: &prototype.TimePointSec{UtcSeconds: resp.State.Dgpo.Time.UtcSeconds + 30}}
	for _, op := range ops {
		tx.AddOperation(op)
	}

	signTx := prototype.SignedTransaction{Trx: tx}

	res := signTx.Sign(privKey, prototype.ChainId{Value: 0})
	signTx.Signature = &prototype.SignatureType{Sig: res}

	if err := signTx.Validate(); err != nil {
		return nil, err
	}

	return &signTx, nil
}
