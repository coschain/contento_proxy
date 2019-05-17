package rpc

import (
	"context"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"google.golang.org/grpc"
	"proxy/config"
	"sync"
	"time"
)

var mutex sync.Mutex

type Client struct {
	alive     bool
	rpcClient grpcpb.ApiServiceClient
	ip        string
	timeout   int
}

type RpcPool struct {
	pool []*Client
}

func NewRpcPool(ips []string, rpcTimeout int) *RpcPool {
	rp := &RpcPool{}

	for _, ip := range ips {
		conn, err := dial(ip)
		if err != nil {
			panic("can not connect to chain rpc")
		}
		rpc := grpcpb.NewApiServiceClient(conn)
		c := &Client{alive: true, rpcClient: rpc, ip: ip, timeout: rpcTimeout}
		rp.push(c)
	}
	go rp.checkAlive()
	return rp
}

func (r *RpcPool) checkAlive() {
	conf := config.GetConfig()
	getAccount := &grpcpb.GetAccountByNameRequest{
		AccountName: &prototype.AccountName{Value: conf.ContractDeployerName},
	}
	for {
		//fmt.Println("@@@ start checkout alive, pool size:",len(r.pool))
		for _, c := range r.pool {
			if c.IsAlive() {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			//start := time.Now()
			_, err := c.rpcClient.GetAccountByName(ctx, getAccount)
			//elapsed := time.Since(start)
			//fmt.Println("---> GetAccountByName took %s", elapsed)
			if err != nil {
				//fmt.Println("@@@ checkAlive set alive false ip:",c.ip)
				c.setAlive(false)
			} else {
				//	fmt.Println("@@@ checkAlive set alive true ip:",c.ip)
				c.setAlive(true)
			}
		}
		time.Sleep(time.Second * 15)
	}
}

func dial(target string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		//logging.VLog().Error("grpcpb.Dial() failed: ", err)
	}
	return conn, err
}

func (r *RpcPool) push(c *Client) {
	r.pool = append(r.pool, c)
}

func (r *RpcPool) GetClient() *Client {
	length := len(r.pool)
	if length == 0 {
		return nil
	}
	mutex.Lock()
	defer mutex.Unlock()

	if r.pool[0].isAlive() {
		//fmt.Println("$$$ GetClient now alive ip:",r.pool[0].ip)
		return r.pool[0]
	} else {
		for i := length - 1; i > 0; i-- {
			if r.pool[i].isAlive() {
				//		fmt.Println("$$$ GetClient change position ip[0]:",r.pool[0].ip," ip[i]:",r.pool[i].ip)
				r.pool[0], r.pool[i] = r.pool[i], r.pool[0]
				return r.pool[0]
			}
		}
		//fmt.Println("$$$ GetClient return nil")
		return r.pool[0] // nothing we can do, nil will cause outer crash
	}
}

func (r *Client) SetAlive(alive bool) {
	mutex.Lock()
	defer mutex.Unlock()
	//fmt.Println("### set alive:",alive," ip:",r.ip)
	r.setAlive(alive)
}

func (r *Client) setAlive(alive bool) {
	r.alive = alive
}

func (r *Client) IsAlive() bool {
	mutex.Lock()
	defer mutex.Unlock()
	return r.isAlive()
}

func (r *Client) isAlive() bool {
	return r.alive
}

func (r *Client) BroadcastTrx(req *grpcpb.BroadcastTrxRequest) (*grpcpb.BroadcastTrxResponse, error) {
	return r.rpcClient.BroadcastTrx(context.Background(), req)
}

func (r *Client) GetUserTrxListByTime(req *grpcpb.GetUserTrxListByTimeRequest) (*grpcpb.GetUserTrxListByTimeResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.timeout)*time.Millisecond)
	defer cancel()
	res, err := r.rpcClient.GetUserTrxListByTime(ctx, req)
	return res, err
}

func (r *Client) GetAccountByName(req *grpcpb.GetAccountByNameRequest) (*grpcpb.AccountResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.timeout)*time.Millisecond)
	defer cancel()
	res, err := r.rpcClient.GetAccountByName(ctx, req)
	return res, err
}

func (r *Client) GetStatisticsInfo(req *grpcpb.NonParamsRequest) (*grpcpb.GetStatResponse, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.timeout)*time.Millisecond)
	defer cancel()
	res, err := r.rpcClient.GetStatisticsInfo(ctx, req)
	return res, err
}

func (r *Client) GetReward(req *grpcpb.GetBlockCashoutRequest) (*grpcpb.BlockCashoutResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.timeout)*time.Millisecond)
	defer cancel()
	res, err := r.rpcClient.GetBlockCashout(ctx, req)
	return res, err
}
