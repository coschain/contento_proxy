package job

import (
	"fmt"
	"github.com/coschain/contentos-go/rpc/pb"
	"proxy/config"
	"proxy/database"
	"proxy/define"
	"proxy/rpc"
	"time"
)

type RewardJob struct {
	//queue chan interface{}
	db        *database.DB
	rpcClient *rpc.Client
}

func NewRewardJob(db *database.DB, pool *rpc.RpcPool) *RewardJob {
	job := &RewardJob{db: db, rpcClient: pool.GetClient()}
	return job
}

func (j *RewardJob) Start() {
	duration := time.Second
	conf := config.GetConfig()
	for {
		time.Sleep(duration)

		height, err := j.getBlockHeight()
		if err != nil {
			continue
		}

		req := &grpcpb.NonParamsRequest{}
		resp, err := j.rpcClient.GetStatisticsInfo(req)
		if err != nil {
			log.Error(fmt.Sprintf("rpc GetStatisticsInfo error:%v", err))
			j.rpcClient.SetAlive(false)
			continue
		}

		irreversibleHeight := resp.State.LastIrreversibleBlockNumber
		//fmt.Println("height:",height," irreversibleHeight:",irreversibleHeight)
		if height == irreversibleHeight {
			//log.Error(fmt.Sprintf("height:%v == irreversibleHeight:%v sleep...",height,irreversibleHeight))
			continue
		}

		if height > irreversibleHeight {
			log.Error(fmt.Sprintf("height:%v > irreversibleHeight:%v chain may be cleaned", height, irreversibleHeight))
			height = 0
		}

		if irreversibleHeight-height > 1 {
			duration = time.Millisecond * time.Duration(conf.RewardMinInterval)
		} else {
			duration = time.Second
		}

		height++
		if j.queryReward(height) {
			j.setBlockHeight(height)
		}
	}
}

func (j *RewardJob) getBlockHeight() (uint64, error) {
	blockHeight, err := j.db.GETUint64(define.BlockHeight)
	if err != nil {
		log.Error(fmt.Sprintf("getBlockHeight error:%v height:%v", err, blockHeight))
	}
	return blockHeight, err
}

func (j *RewardJob) setBlockHeight(blockHeight uint64) {
	if err := j.db.SET(define.BlockHeight, blockHeight); err != nil {
		log.Error(fmt.Sprintf("setBlockHeight error:%v height:%v", err, blockHeight))
	}
}

func (j *RewardJob) queryReward(height uint64) bool {
	req := &grpcpb.GetBlockCashoutRequest{
		BlockHeight: height,
	}

	resp, err := j.rpcClient.GetReward(req)
	if err != nil {
		log.Error(fmt.Sprintf("rpc GetReward error:%v", err))
		j.rpcClient.SetAlive(false)
		return false
	}

	if len(resp.CashoutList) == 0 {
		return true
	}

	for _, cash := range resp.CashoutList {
		id, err := j.db.GETId(cash.AccountName.Value)
		if err != nil {
			log.Error(fmt.Sprintf("GETId name:%v error:%v", cash.AccountName.Value, err))
			continue
		}
		if id == "" {
			//log.Warn(fmt.Sprintf("GETId name:%v empty",cash.AccountName.Value))
			continue
		}

		if err := j.db.AddReward(id, define.Reward, cash.Reward.Value); err != nil {
			log.Error(fmt.Sprintf("AddReward error:%v id:%v", err, id))
			continue
		} else {
			log.Info(fmt.Sprintf("AddReward ok id:%v reward:%v", id, cash.Reward.Value))
		}
	}
	return true
}
