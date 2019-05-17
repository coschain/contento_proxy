package server

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"net"
	"net/http"
	"os"
	"proxy/config"
	"proxy/database"
	"proxy/job"
	"proxy/rpc"
	"strings"
	"time"
)

const (
	httpReadTimeout  = 5 //seconds
	httpWriteTimeout = 5 //seconds
)

const (
	OK                 = 1000
	ParamError         = 2000
	ServerError        = 2001
	IdDuplicate        = 3000
	IdNotExist         = 3001
	PostIdDuplicate    = 3002
	PostIdNotExist     = 3003
	CommentIdDuplicate = 3004
	CommentIdNotExist  = 3005
	FuidNotExist       = 3006
	LikePostDuplicate  = 3007
	FollowDuplicate    = 3008
	FollowSelf         = 3009
	Signed             = 3010
	GameIdExist        = 3011
)

const (
	PhotoGrid = 1
	Contentos = 2
	Game2048  = 3
)

var (
	httpListener net.Listener
	closed       bool
	dbInstance   *database.DB
	jobs         []*job.Job
	rJob         *job.RewardJob
	log          *logrus.Logger
	jobCount     int
	limiter      *rate.Limiter
)

func init() {
	log = logrus.New()
}

// Init init the http module.
func Init(db *database.DB, apiLogFile *os.File, jobLogFile *os.File) error {
	if apiLogFile == nil {
		panic("log file is nil")
	}
	log.Out = apiLogFile
	log.SetReportCaller(true)

	dbInstance = db
	conf := config.GetConfig()
	jobCount = conf.JobCount

	pool := rpc.NewRpcPool(conf.RpcAddr, conf.RpcTimeOut)
	for i := 0; i < jobCount; i++ {
		jobInstance := job.NewJob(db, jobLogFile, i, pool)
		jobs = append(jobs, jobInstance)
		go jobInstance.Start()
	}

	// reward query job
	rJob = job.NewRewardJob(db, pool)
	go rJob.Start()

	// rate limiter
	limiter = rate.NewLimiter(rate.Limit(conf.TokenPerSecond), conf.TokenMax)

	// init handler
	httpServeMux := initHttpHandler()
	server := &http.Server{Handler: httpServeMux, ReadTimeout: httpReadTimeout * time.Second, WriteTimeout: httpWriteTimeout * time.Second}

	addr := conf.ListenAddr
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("tls.Listen(\"tcp\", \"%s\") error(%v)", addr, err)
		return err
	}
	httpListener = l
	go func() {
		log.Info("start http listen addr: %s", addr)
		if err := server.Serve(l); err != nil {
			log.Error("server.Serve(\"%s\") error(%v)", addr, err)
			if !closed {
				panic(err)
			}
		}
	}()

	return nil
}

func checkLimit(w http.ResponseWriter, r *http.Request) bool {
	if limiter.Allow() == false {
		http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
		return false
	}
	return true
}

// initHttpHandler register all controller http handlers.
func initHttpHandler() *http.ServeMux {
	httpServeMux := http.NewServeMux()
	httpServeMux.HandleFunc("/api/game2048", func(w http.ResponseWriter, r *http.Request) {
		if !checkLimit(w, r) {
			return
		}
		game2048(w, r)
	})
	httpServeMux.HandleFunc("/api/actionlist", func(w http.ResponseWriter, r *http.Request) {
		if !checkLimit(w, r) {
			return
		}
		actionList(w, r)
	})
	httpServeMux.HandleFunc("/api/signin", func(w http.ResponseWriter, r *http.Request) {
		if !checkLimit(w, r) {
			return
		}
		signIn(w, r)
	})
	httpServeMux.HandleFunc("/api/account", func(w http.ResponseWriter, r *http.Request) {
		if !checkLimit(w, r) {
			return
		}
		createAccount(w, r)
	})
	httpServeMux.HandleFunc("/api/post", func(w http.ResponseWriter, r *http.Request) {
		if !checkLimit(w, r) {
			return
		}
		post(w, r)
	})
	httpServeMux.HandleFunc("/api/like", func(w http.ResponseWriter, r *http.Request) {
		if !checkLimit(w, r) {
			return
		}
		like(w, r)
	})
	httpServeMux.HandleFunc("/api/comment", func(w http.ResponseWriter, r *http.Request) {
		if !checkLimit(w, r) {
			return
		}
		comment(w, r)
	})
	httpServeMux.HandleFunc("/api/follow", func(w http.ResponseWriter, r *http.Request) {
		if !checkLimit(w, r) {
			return
		}
		follow(w, r)
	})
	httpServeMux.HandleFunc("/api/unfollow", func(w http.ResponseWriter, r *http.Request) {
		if !checkLimit(w, r) {
			return
		}
		unfollow(w, r)
	})
	httpServeMux.HandleFunc("/api/getname", func(w http.ResponseWriter, r *http.Request) {
		getName(w, r)
	})
	return httpServeMux
}

// retGetWriter is a json writer for http get method.
func retGetWriter(r *http.Request, wr http.ResponseWriter, start time.Time, result map[string]interface{}) {
	wr.Header().Set("Content-Type", "application/json;charset=utf-8")
	ret := result["ret"].(int)
	byteJson, err := json.Marshal(result)
	if err != nil {
		log.Error("json.Marshal(\"%v\") failed (%v)", result, err)
		return
	}
	if _, err := wr.Write(byteJson); err != nil {
		log.Error("wr.Write(\"%s\") failed (%v)", string(byteJson), err)
		return
	}
	ip := getClientIp(r)
	now := time.Now()
	hs := now.Sub(start)
	logStr := fmt.Sprintf("[%v] get:%v time:%v ret:%v)", ip, r.URL.String(), hs.Seconds(), ret)
	log.Info(logStr)
}

// retPostWriter is a json writer for http post method.
func retPostWriter(r *http.Request, wr http.ResponseWriter, params *string, start time.Time, result map[string]interface{}) {
	wr.Header().Set("Content-Type", "application/json;charset=utf-8")
	ret := result["ret"].(int)
	byteJson, err := json.Marshal(result)
	if err != nil {
		log.Error("json.Marshal(\"%v\") failed (%v)", result, err)
		return
	}
	if _, err := wr.Write(byteJson); err != nil {
		log.Error("wr.Write(\"%s\") failed (%v)", string(byteJson), err)
		return
	}
	ip := getClientIp(r)
	hs := time.Now().Sub(start)
	logStr := fmt.Sprintf("[%v] post_url:%v param:%v,time:%v,ret:%v)", ip, r.URL.String(), *params, hs.Seconds(), ret)
	log.Info(logStr)
}

// get client ip from http request
func getClientIp(r *http.Request) string {
	remote := r.Header.Get("X-Forwarded-For")
	idx := strings.Index(remote, ",")
	if remote != "" && idx > -1 {
		remote = remote[:idx]
	} else if remote == "" {
		remote = r.Header.Get("X-Real-IP")
	}
	return remote
}

// Close close the resource.
func Close() {
	closed = true
	if err := httpListener.Close(); err != nil {
		log.Error("l.Close() error(%v)", err)
	}
	httpListener = nil
}
