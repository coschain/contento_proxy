package job

import (
	"fmt"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"proxy/config"
	"proxy/define"
	"proxy/utils"
	"strconv"
)

/**
 * 用户之间转账
 */
type LoserTransferWinnerOption struct {
	Lid      string // 失败者uid
	Lname    string // 失败者名字
	Lprivkey string // 失败者私钥
	Wname    string // 获胜者名字
	Cosnum   uint64 // 需要转账的cos数量
	Memo     string // 转账说明
}

func (j *Job) loserTransferToWinner(option *LoserTransferWinnerOption) bool {

	memo := utils.GenerateUUID(option.Lname)
	transOp := &prototype.TransferOperation{
		From:   &prototype.AccountName{Value: option.Lname},
		To:     &prototype.AccountName{Value: option.Wname},
		Amount: &prototype.Coin{Value: option.Cosnum},
		Memo:   strconv.Itoa(int(memo)),
	}
	signTx1, err := utils.GenerateSignedTx(option.Lprivkey, j.rpcPool.GetClient(), transOp)
	if err != nil {
		log.Error(fmt.Sprintf("GenerateSignedTx error:%v", err))
		return false
	}
	if !j.call(option.Lid, option.Lname, "loserTransferToWinner", signTx1) {
		log.Error(fmt.Sprintf("loserTransferToWinner error:%v", err))
		return false
	}
	log.Info(fmt.Sprintf("loserTransferToWinnerInfo: transOp:%v", transOp))
	return true
}

/**
 * 默认官号赠送cos
 *   官方默认赠送是为了程序正常执行需要
 *   如果一个账号的剩余cos数量超过0.3cos，将不再赠送
 */
type TransferOption struct {
	id    string // 用户id
	name  string // 用户昵称
	val   uint64 // 赠送数量
	limit uint64 // 限制（用户cos数量超过多少之后，不给赠送）
}

func (j *Job) transferTo(option *TransferOption) bool {

	if info, err := j.getAccountInfo(option.name); err != false && info != nil && option.limit > 0 {
		if coin := info.GetInfo().GetCoin().GetValue(); coin >= option.limit {
			log.Info(fmt.Sprintf("id:%v, name:%v, coin:%v, limit:%v", option.id, option.name, coin, option.limit))
			return true
		}
	}

	conf := config.GetConfig()
	transOp := &prototype.TransferOperation{
		From:   &prototype.AccountName{Value: conf.TransferName},
		To:     &prototype.AccountName{Value: option.name},
		Amount: &prototype.Coin{Value: option.val},
	}
	signTx1, err := utils.GenerateSignedTx(conf.TransferPriKey, j.rpcPool.GetClient(), transOp)
	if err != nil {
		log.Error(fmt.Sprintf("GenerateSignedTx error:%v", err))
		return false
	}
	if !j.call(option.id, conf.TransferName, "transfer", signTx1) {
		log.Error(fmt.Sprintf("transfer error:%v", err))
		return false
	}
	return true
}

func (j *Job) getAccountInfo(accountName string) (*grpcpb.AccountResponse, bool) {

	// query account if exists in chain
	getAccount := &grpcpb.GetAccountByNameRequest{
		AccountName: &prototype.AccountName{Value: accountName},
	}
	c := j.rpcPool.GetClient()
	resp, err := c.GetAccountByName(getAccount)
	if err != nil {
		log.Error(fmt.Sprintf("rpc GetAccountByName error:%v, accountName:%v", err, accountName))
		c.SetAlive(false)
		return nil, false
	}
	return resp, true
}

func (j *Job) accountExistInChain(accountName string) bool {
	// query account if exists in chain
	getAccount := &grpcpb.GetAccountByNameRequest{
		AccountName: &prototype.AccountName{Value: accountName},
	}
	c := j.rpcPool.GetClient()
	resp, err := c.GetAccountByName(getAccount)
	if err != nil {
		log.Error(fmt.Sprintf("rpc GetAccountByName error:%v", err))
		c.SetAlive(false)
		return false
	}
	if resp.Info.AccountName != nil {
		log.Info(fmt.Sprintf("rpc GetAccountByName account still on the chain:%v", resp.Info.AccountName))
		return true
	}
	return false
}

func (j *Job) call(uid, name, opType string, signTx *prototype.SignedTransaction) bool {
	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	c := j.rpcPool.GetClient()
	id, _ := signTx.Id()
	idStr := fmt.Sprintf("%x", id.Hash)
	res, err := c.BroadcastTrx(req)
	if err != nil || res == nil {
		log.Error(fmt.Sprintf("job_%v broadcast id:%v name:%v op:%v error:%v res:%v hash:%v", j.index, uid, name, opType, err, res, idStr))
		c.SetAlive(false)
		return false
	} else {
		if res.Invoice.Status != 200 {
			log.Error(fmt.Sprintf("job_%v broadcast id:%v name:%v op:%v res status error:%v hash:%v res:%v", j.index, uid, name, opType, err, idStr, res))
			return false
		}
		log.Info(fmt.Sprintf("job_%v broadcast id:%v name:%v op:%v response:%v hash:%v", j.index, uid, name, opType, res, idStr))
		return true
	}
}

func (j *Job) callContract(id, name, opName, contract, method, param string) bool {
	conf := config.GetConfig()
	applyOp := &prototype.ContractApplyOperation{
		Caller:   &prototype.AccountName{Value: name},
		Owner:    &prototype.AccountName{Value: conf.ContractDeployerName},
		Amount:   &prototype.Coin{Value: 0},
		Gas:      &prototype.Coin{Value: 300000},
		Contract: contract,
		Params:   param,
		Method:   method,
	}

	privKeyStr, err := j.db.HGETString(id, define.PrivateKey)
	if err != nil || privKeyStr == "" {
		log.Error(fmt.Sprintf("get private key error:%v key:%v", err, privKeyStr))
		return false
	}

	signTx, err := utils.GenerateSignedTx(privKeyStr, j.rpcPool.GetClient(), applyOp)
	if err != nil {
		log.Error(fmt.Sprintf("GenerateSignedTx error:%v", err))
		return false
	}
	return j.call(id, name, opName, signTx)
}

func (j *Job) createAccount(id, name, app string) bool {
	// generate prikey and pubkey
	pubKeyStr, privKeyStr, err := utils.GenerateNewKey()

	// we just record info in proxy,if chain failed, we can repair chain via info when subsequent PG's request come
	if err := j.db.SetAccount(id, define.Name, name, define.PubKey, pubKeyStr, define.PrivateKey, privKeyStr); err != nil {
		log.Error(fmt.Sprintf("SetAccount error:%v", err))
		return false
	}

	conf := config.GetConfig()
	creator := conf.CreatorMap[app]
	if creator == nil {
		log.Error(fmt.Sprintf("can not found creator for:%v", app))
		return false
	}
	// write to chain
	pubkey, _ := prototype.PublicKeyFromWIF(pubKeyStr)
	acop := &prototype.AccountCreateOperation{
		Fee:            prototype.NewCoin(1),
		Creator:        &prototype.AccountName{Value: creator.CreatorName},
		NewAccountName: &prototype.AccountName{Value: name},
		Owner:          pubkey,
	}
	signTx, err := utils.GenerateSignedTx(creator.CreatorPriKey, j.rpcPool.GetClient(), acop)
	if err != nil {
		log.Error(fmt.Sprintf("GenerateSignedTx error:%v", err))
		return false
	}
	return j.call(id, name, "accountcreate", signTx)
}

func (j *Job) GetUserActionList(accountName string) (*grpcpb.GetUserTrxListByTimeResponse, bool) {
	getList := &grpcpb.GetUserTrxListByTimeRequest{
		Name:    &prototype.AccountName{Value: accountName},
		Start:   nil,
		End:     nil,
		LastTrx: nil,
		Limit:   5,
	}
	c := j.rpcPool.GetClient()
	resp, err := c.GetUserTrxListByTime(getList)
	if err != nil {
		log.Error(fmt.Sprintf("rpc GetUserTrxListByTime error:%v", err))
		c.SetAlive(false)
		return nil, false
	}

	return resp, true
}
