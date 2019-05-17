package job

import (
	"github.com/sirupsen/logrus"
	"os"
	"proxy/database"
	"proxy/rpc"
)

const (
	size    = 500
	tfLimit = uint64(300000)
)

type Trace struct {
	AppStr string
}

type Job struct {
	index   int
	queue   chan interface{}
	db      *database.DB
	rpcPool *rpc.RpcPool
}

var log *logrus.Logger

func init() {
	log = logrus.New()
}

func NewJob(db *database.DB, f *os.File, i int, pool *rpc.RpcPool) *Job {
	if f == nil {
		panic("job's log file is nil")
	}
	log.Out = f
	log.SetReportCaller(true)

	job := &Job{queue: make(chan interface{}, size)}
	job.rpcPool = pool
	job.db = db
	job.index = i
	return job
}

func (j *Job) Start() {
	for {
		msg := <-j.queue
		switch x := msg.(type) {
		case *AccountMsg:
			j.processAccountMsg(x)
		case *PostMsg:
			j.processPostMsg(x)
		case *LikeMsg:
			j.processLikeMsg(x)
		case *CommentMsg:
			j.processCommentMsg(x)
		case *FollowMsg:
			j.processFollowMsg(x)
		case *FakeLikeMsg:
			j.processFakeLikeMsg(x)
		case *FakeCommentMsg:
			j.processFakeCommentMsg(x)
		case *SignInMsg:
			j.processSignInMsg(x)
		case *Game2048Msg:
			j.processGame2048Msg(x)
		default:
		}
	}
}

func (j *Job) Put(m interface{}) {
	j.queue <- m
}
