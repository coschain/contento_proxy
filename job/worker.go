package job

import (
	"encoding/json"
	"fmt"
	"github.com/coschain/contentos-go/prototype"
	"math/rand"
	"proxy/config"
	"proxy/define"
	"proxy/utils"
)

/**
 * 创建账号
 */
type AccountMsg struct {
	Trace
	Id   string
	Name string
}

/**
 * 发表评论
 */
type CommentMsg struct {
	Trace
	Id        string
	PostId    string
	CommentId string
	Content   string
}

/**
 * 关注
 */
type FollowMsg struct {
	Trace
	Uid          string
	Fuid         string
	UniqueFollow string
	Cancel       bool
}

/**
 * 签到
 */
type SignInMsg struct {
	Trace
	Id   string
	Date string
}

/**
 * 2048游戏上报
 */
type Game2048Msg struct {
	Trace
	Wcos  uint64 // 获胜者所拥有的cos数量（ 2048上报 ）
	Lcos  uint64 // 失败者所拥有的cos数量（ 2048上报 ）
	Cos   uint64 // 本次比赛下注数量（ 失败者需要转移给获胜者多少cos ）
	Wid   string // 获胜者uid
	Wname string // 获取中名字
	Lid   string // 失败者uid
	Lname string // 失败者名字
	Gid   string // 比赛id
}

/**
 * PG发帖 / Contentos上传视频
 */
type PostMsg struct {
	Trace
	Id      string
	PostId  string
	Title   string
	Content string
	Tag     string
}

type FakeCommentMsg struct {
	Trace
	Id      string
	Name    string
	Content string
}

/**
 * 点赞
 */
type LikeMsg struct {
	Trace
	Id         string
	PostId     string
	UniqueName string
}

type FakeLikeMsg struct {
	Trace
	Id   string
	Name string
}

func (j *Job) processGame2048Msg(m *Game2048Msg) {

	// check unique
	if err := j.db.SET(m.Gid, 1); err != nil {
		return
	}

	// get name
	winnerName, err := j.db.HGETString(m.Wid, define.Name)
	if err != nil {
		log.Error(fmt.Sprintf("Get winnerName error: wid:%v, lid:%v, gid:%v", m.Wid, m.Lid, m.Gid))
		return
	}
	if winnerName != "" {
		m.Wname = winnerName
	}

	loserName, err := j.db.HGETString(m.Lid, define.Name)
	if err != nil {
		log.Error(fmt.Sprintf("Get winnerName error: wid:%v, lid:%v, gid:%v", m.Wid, m.Lid, m.Gid))
		return
	}
	if loserName != "" {
		m.Lname = loserName
	}

	conf := config.GetConfig()
	// if account not exist in chain, create if firstly
	winnerAccountInfo, werr := j.getAccountInfo(m.Wname)
	if werr == false || winnerAccountInfo == nil {
		if !j.createAccount(m.Wid, m.Wname, m.AppStr) {
			log.Error(fmt.Sprintf("repair account:%v name:%v failed", m.Wid, m.Wname))
			return
		} else {
			// transfer to cos
			if !j.transferTo(&TransferOption{id: m.Wid, name: m.Wname, val: m.Wcos, limit: 0}) {
				return
			}
		}
	} else {
		// check coin
		getWinnerCoin := winnerAccountInfo.GetInfo().GetCoin().GetValue()
		// 2048 cos > chain cos
		if getWinnerCoin < (m.Wcos - m.Cos) {
			if !j.transferTo(&TransferOption{id: m.Wid, name: m.Wname, val: (m.Wcos - m.Cos - getWinnerCoin), limit: 0}) {
				return
			}

			// chain cos > 2048 cos
		} else if getWinnerCoin > (m.Wcos - m.Cos) {
			WprivKeyStr, err := j.db.HGETString(m.Wid, define.PrivateKey)
			if err != nil || WprivKeyStr == "" {
				log.Error(fmt.Sprintf("get private key error:%v account:%v key:%v", err, m.Wid, WprivKeyStr))
				return
			}
			if !j.loserTransferToWinner(&LoserTransferWinnerOption{
				Lid: m.Wid, Lname: m.Wname, Lprivkey: WprivKeyStr,
				Wname:  conf.TransferName,
				Cosnum: getWinnerCoin - m.Wcos + m.Cos,
				Memo:   ""},
			) {
				log.Error(fmt.Sprintf("tarnsfer error, id:%v name:%v, cos:%v", m.Wid, m.Wname, getWinnerCoin))
				return
			}
		}
	}

	// loser
	accountInfo, aerr := j.getAccountInfo(m.Lname)
	if aerr == false || accountInfo == nil {
		if !j.createAccount(m.Lid, m.Lname, m.AppStr) {
			log.Error(fmt.Sprintf("repair account:%v name:%v failed", m.Lid, m.Lname))
			return
		} else {
			// transfer to cos
			if !j.transferTo(&TransferOption{id: m.Lid, name: m.Lname, val: m.Lcos, limit: 0}) {
				return
			}
		}
	} else {
		// check coin
		getCoin := accountInfo.GetInfo().GetCoin().GetValue()
		if getCoin < (m.Lcos + m.Cos) {
			if !j.transferTo(&TransferOption{id: m.Lid, name: m.Lname, val: (m.Lcos - getCoin + m.Cos), limit: 0}) {
				return
			}
		} else if getCoin > (m.Lcos + m.Cos) {

			// transfer
			LprivKeyStr, err := j.db.HGETString(m.Lid, define.PrivateKey)
			if err != nil || LprivKeyStr == "" {
				log.Error(fmt.Sprintf("get private key error:%v account:%v key:%v", err, m.Lid, LprivKeyStr))
				return
			}
			if !j.loserTransferToWinner(&LoserTransferWinnerOption{
				Lid: m.Lid, Lname: m.Lname, Lprivkey: LprivKeyStr,
				Wname:  conf.TransferName,
				Cosnum: getCoin - m.Lcos + m.Cos,
				Memo:   ""},
			) {
				log.Error(fmt.Sprintf("tarnsfer error, id:%v name:%v, cos:%v", m.Lid, m.Lname, getCoin))
				return
			}

		}
	}

	// transfer
	privKeyStr, err := j.db.HGETString(m.Lid, define.PrivateKey)
	if err != nil || privKeyStr == "" {
		log.Error(fmt.Sprintf("get private key error:%v account:%v key:%v", err, m.Lid, privKeyStr))
		return
	}
	memo := ""
	if !j.loserTransferToWinner(
		&LoserTransferWinnerOption{
			Lid: m.Lid, Lname: m.Lname, Lprivkey: privKeyStr,
			Wname:  m.Wname,
			Cosnum: m.Cos,
			Memo:   memo,
		},
	) {
		log.Error(fmt.Sprintf("tarnsfer error, loserId:%v, loserName:%v, winnerName:%v, cosNum:%v", m.Lid, m.Lname, m.Wname, m.Cos))
		return
	}
}

func (j *Job) processFakeCommentMsg(m *FakeCommentMsg) {
	// repair chain's account
	if !j.accountExistInChain(m.Name) {
		if !j.createAccount(m.Id, m.Name, m.AppStr) {
			log.Error(fmt.Sprintf("repair account:%v failed", m.Name))
			return
		}
	}

	if !j.transferTo(
		&TransferOption{id: m.Id, name: m.Name, val: tfLimit, limit: tfLimit},
	) {
		return
	}

	conf := config.GetConfig()
	commentUUID := utils.GenerateUUID(m.Name)
	randomNum := uint64(rand.Uint32())<<32 + uint64(rand.Uint32())
	content, err := json.Marshal(m.Content)
	if err != nil {
		log.Error(fmt.Sprintf("json encode error: id:%v, name:%v, method:%v, content:%v", m.Id, m.Name, "FakeComment", m.Content))
	}
	param := fmt.Sprintf(" [%v,\"%v\",%v,%v] ", commentUUID, m.Name, string(content), randomNum)
	j.callContract(m.Id, m.Name, "FakeComment", conf.ContractName, conf.ContractCommentMethod, param)
}

func (j *Job) processFakeLikeMsg(m *FakeLikeMsg) {
	// repair chain's account
	if !j.accountExistInChain(m.Name) {
		if !j.createAccount(m.Id, m.Name, m.AppStr) {
			log.Error(fmt.Sprintf("repair account:%v failed", m.Name))
			return
		}
	}

	if !j.transferTo(
		&TransferOption{id: m.Id, name: m.Name, val: tfLimit, limit: tfLimit},
	) {
		return
	}

	conf := config.GetConfig()
	randomNum := uint64(rand.Uint32())<<32 + uint64(rand.Uint32())
	param := fmt.Sprintf(" [\"%v\",%v] ", m.Name, randomNum)
	j.callContract(m.Id, m.Name, "fakeLike", conf.ContractName, conf.ContractLikeMethod, param)
}

func (j *Job) createPost(pid, id, title, content, tag, app string) bool {
	name, err := j.db.HGETString(id, define.Name)
	if err != nil || name == "" {
		log.Error(fmt.Sprintf("get account name error:%v account:%v name:%v", err, id, name))
		return false
	}

	uuid := utils.GenerateUUID(name + title)
	// we just record info in proxy,if chain failed, we can repair chain via info when subsequent PG's request come
	if err := j.db.SetPostInfo(pid, define.UUID, uuid, define.Owner, id, define.ParentId, 0); err != nil {
		log.Error(fmt.Sprintf("SetPostInfo error:%v post_id:%v", err, pid))
		return false
	}

	// if account not exist in chain, create it firstly
	if !j.accountExistInChain(name) {
		if !j.createAccount(id, name, app) {
			log.Error(fmt.Sprintf("repair account:%v failed", name))
			return false
		}
	}
	privKeyStr, err := j.db.HGETString(id, define.PrivateKey)
	if err != nil || privKeyStr == "" {
		log.Error(fmt.Sprintf("get private key error:%v key:%v", err, privKeyStr))
		return false
	}

	// write to chain
	postOp := &prototype.PostOperation{
		Uuid:          uuid,
		Owner:         &prototype.AccountName{Value: name},
		Title:         title,
		Content:       content,
		Tags:          []string{tag},
		Beneficiaries: []*prototype.BeneficiaryRouteType{},
	}
	postOp.Beneficiaries = append(postOp.Beneficiaries, &prototype.BeneficiaryRouteType{
		Name:   &prototype.AccountName{Value: name},
		Weight: 1,
	})
	signTx, err := utils.GenerateSignedTx(privKeyStr, j.rpcPool.GetClient(), postOp)
	if err != nil {
		log.Error(fmt.Sprintf("GenerateSignedTx error:%v", err))
		return false
	}
	return j.call(id, name, "post", signTx)
}

func (j *Job) processSignInMsg(m *SignInMsg) {

	// check the uInfo
	name, err := j.db.HGETString(m.Id, define.Name)
	if err != nil || name == "" {
		log.Error(fmt.Sprintf("get account name error:%v account:%v name:%v", err, m.Id, name))
		return
	}

	// check unique
	uniqueKey := m.Id + m.Date
	if err := j.db.SET(uniqueKey, 1); err != nil {
		return
	}
	// if account not exist in chain, create it firstly
	if !j.accountExistInChain(name) {
		if !j.createAccount(m.Id, name, m.AppStr) {
			log.Error(fmt.Sprintf("repair account:%v name:%v failed", m.Id, name))
			return
		}
	}

	if !j.transferTo(
		&TransferOption{id: m.Id, name: name, val: tfLimit, limit: tfLimit},
	) {
		return
	}

	// 上报
	conf := config.GetConfig()
	param := fmt.Sprintf(" [\"%v\"] ", name)
	j.callContract(m.Id, name, "signIn", conf.ContractName, conf.ContractSignInMethod, param)
}

func (j *Job) processLikeMsg(m *LikeMsg) {
	name, err := j.db.HGETString(m.Id, define.Name)
	if err != nil || name == "" {
		log.Error(fmt.Sprintf("get account name error:%v account:%v name:%v", err, m.Id, name))
		return
	}

	// write unique like -> post
	if err := j.db.SET(m.UniqueName, 1); err != nil {
		return
	}

	// if account not exist in chain, create it firstly
	if !j.accountExistInChain(name) {
		if !j.createAccount(m.Id, name, m.AppStr) {
			log.Error(fmt.Sprintf("repair account:%v name:%v failed", m.Id, name))
			return
		}
	}
	privKeyStr, err := j.db.HGETString(m.Id, define.PrivateKey)
	if err != nil || privKeyStr == "" {
		log.Error(fmt.Sprintf("get private key error:%v account:%v key:%v", err, m.Id, privKeyStr))
		return
	}

	owner, err := j.db.HGETString(m.PostId, define.Owner)
	if err != nil {
		log.Error(fmt.Sprintf("get post:%v owner failed", m.PostId))
		return
	}

	uuid, err := j.db.HGETUint64(m.PostId, define.UUID)
	if err != nil {
		log.Error(fmt.Sprintf("get post:%v uuid failed", m.PostId))
		return
	}
	ownerExist, errOwner := j.db.EXISTS(owner)
	if errOwner != nil || !ownerExist {
		log.Error(fmt.Sprintf("get post:%v owner:v failed", m.PostId, owner))
		return
	}
	ownerName, err := j.db.HGETString(owner, define.Name)
	if err != nil || ownerName == "" {
		log.Error(fmt.Sprintf("get account name error:%v name:%v", err, ownerName))
		return
	}
	// if owner not exist in chain, create it firstly
	if !j.accountExistInChain(ownerName) {
		if !j.createAccount(owner, ownerName, m.AppStr) {
			log.Error(fmt.Sprintf("repair account:%v failed", ownerName))
			return
		}
	}

	// like
	likeOp := &prototype.VoteOperation{
		Voter: prototype.NewAccountName(name),
		Idx:   uuid,
	}

	signTx, err := utils.GenerateSignedTx(privKeyStr, j.rpcPool.GetClient(), likeOp)
	if err != nil {
		log.Error(fmt.Sprintf("GenerateSignedTx error:%v", err))
		return
	}
	j.call(m.Id, name, "vote", signTx)
}

func (j *Job) processCommentMsg(m *CommentMsg) {
	name, err := j.db.HGETString(m.Id, define.Name)
	if err != nil || name == "" {
		log.Error(fmt.Sprintf("get account name error:%v account:%v name:%v", err, m.Id, name))
		return
	}

	commentUUID := utils.GenerateUUID(name)
	if err := j.db.SetPostInfo(m.CommentId, define.UUID, commentUUID, define.Owner, m.Id, define.ParentId, m.PostId); err != nil {
		log.Error(fmt.Sprintf("SetPostInfo error:%v comment id:%v", err, m.CommentId))
		return
	}

	// if account not exist in chain, create it firstly
	if !j.accountExistInChain(name) {
		if !j.createAccount(m.Id, name, m.AppStr) {
			log.Error(fmt.Sprintf("repair account:%v failed", name))
			return
		}
	}
	privKeyStr, err := j.db.HGETString(m.Id, define.PrivateKey)
	if err != nil || privKeyStr == "" {
		log.Error(fmt.Sprintf("get account:%v private key failed", m.Id))
		return
	}

	owner, err := j.db.HGETString(m.PostId, define.Owner)
	if err != nil {
		log.Error(fmt.Sprintf("get post:%v owner failed", m.PostId))
		return
	}

	postUUID, err := j.db.HGETUint64(m.PostId, define.UUID)
	if err != nil {
		log.Error(fmt.Sprintf("get post:%v uuid failed", m.PostId))
		return
	}

	// find owner of post
	ownerName, err := j.db.HGETString(owner, define.Name)
	if err != nil || ownerName == "" {
		log.Error(fmt.Sprintf("get account name error:%v account:%v name:%v", err, owner, ownerName))
		return
	}
	// if owner not exist in chain, create it firstly
	if !j.accountExistInChain(ownerName) {
		if !j.createAccount(owner, ownerName, m.AppStr) {
			log.Error(fmt.Sprintf("repair account:%v failed", ownerName))
			return
		}
	}

	// comment
	commentOp := &prototype.ReplyOperation{
		Uuid:          commentUUID,
		Owner:         prototype.NewAccountName(name),
		Content:       m.Content,
		ParentUuid:    postUUID,
		Beneficiaries: []*prototype.BeneficiaryRouteType{},
	}
	commentOp.Beneficiaries = append(commentOp.Beneficiaries, &prototype.BeneficiaryRouteType{
		Name:   &prototype.AccountName{Value: name},
		Weight: 1,
	})

	signTx, err := utils.GenerateSignedTx(privKeyStr, j.rpcPool.GetClient(), commentOp)
	if err != nil {
		log.Error(fmt.Sprintf("GenerateSignedTx error:%v", err))
		return
	}
	j.call(m.Id, name, "reply", signTx)
}

func (j *Job) processFollowMsg(m *FollowMsg) {
	uidName, err := j.db.HGETString(m.Uid, define.Name)
	if err != nil || uidName == "" {
		log.Error(fmt.Sprintf("get account name account:%v error:%v name:%v", err, m.Uid, uidName))
		return
	}
	fUidName, err := j.db.HGETString(m.Fuid, define.Name)
	if err != nil || fUidName == "" {
		log.Error(fmt.Sprintf("get account name error:%v account:%v name:%v", err, m.Fuid, fUidName))
		return
	}

	// update unique follow unfollow
	if m.Cancel {
		if err := j.db.DEL(m.UniqueFollow); err != nil {
			return
		}
	} else {
		if err := j.db.SET(m.UniqueFollow, 1); err != nil {
			return
		}
	}

	// if account not exist in chain, create it firstly
	if !j.accountExistInChain(uidName) {
		if !j.createAccount(m.Uid, uidName, m.AppStr) {
			log.Error(fmt.Sprintf("repair account:%v name:%v failed", m.Uid, uidName))
			return
		}
	}
	// if followed account not exist in chain, create it firstly
	if !j.accountExistInChain(fUidName) {
		if !j.createAccount(m.Fuid, fUidName, m.AppStr) {
			log.Error(fmt.Sprintf("repair account:%v name:%v failed", m.Fuid, fUidName))
			return
		}
	}

	privKeyStr, err := j.db.HGETString(m.Uid, define.PrivateKey)
	if err != nil || privKeyStr == "" {
		log.Error(fmt.Sprintf("get account:%v private key failed", m.Uid))
		return
	}

	// follow
	followOp := &prototype.FollowOperation{
		Account:  prototype.NewAccountName(uidName),
		FAccount: prototype.NewAccountName(fUidName),
		Cancel:   m.Cancel,
	}

	signTx, err := utils.GenerateSignedTx(privKeyStr, j.rpcPool.GetClient(), followOp)
	if err != nil {
		log.Error(fmt.Sprintf("GenerateSignedTx error:%v", err))
		return
	}
	opStr := "follow"
	if m.Cancel {
		opStr = "unfollow"
	}
	j.call(m.Uid, uidName, opStr, signTx)
}

func (j *Job) processAccountMsg(m *AccountMsg) {
	j.createAccount(m.Id, m.Name, m.AppStr)
}

func (j *Job) processPostMsg(m *PostMsg) {
	j.createPost(m.PostId, m.Id, m.Title, m.Content, m.Tag, m.AppStr)
}
