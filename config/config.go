package config

import (
	"bufio"
	"github.com/jinzhu/configor"
	"os"
	"proxy/define"
	"strings"
	"sync"
)

type Creator struct {
	CreatorName   string
	CreatorPriKey string
}

type Config struct {
	ListenAddr       string   `default:"0.0.0.0:8000"`
	RpcAddr          []string `default:""`
	RedisAddr        string   `default:""`
	ApiLogPath       string   `default:""`
	JobLogPath       string   `default:""`
	RedisMaxIdle     int      `default:"30"`
	RedisMaxActive   int      `default:"100"`
	RedisIdleTimeout int      `default:"30"`
	JobCount         int      `default:"100"`
	TokenPerSecond   int      `default:"1000"`
	TokenMax         int      `default:"1500"`
	Creators         []struct {
		Type          string `default:""`
		CreatorName   string `default:""`
		CreatorPriKey string `default:""`
	}
	CreatorMap            map[string]*Creator
	ContractDeployerName  string `default:""`
	RpcTimeOut            int    `default:"350"`
	ContractName          string `default:""`
	ContractCommentMethod string `default:""`
	ContractSignInMethod  string `default:""`
	ContractLikeMethod    string `default:""`
	RewardMinInterval     int    `default:"100"`
	TransferName          string `default:""`
	TransferPriKey        string `default:""`
}

var once sync.Once
var c *Config

func GetConfig() *Config {
	once.Do(func() {

		// default and online config name
		configName := "config.yml"

		// get idc
		fb, err := os.Open("/data/app/idc/go-idc.ini")
		defer fb.Close()
		if err == nil {
			rd := bufio.NewReader(fb)
			idc, err := rd.ReadString('\n')
			if err == nil {
				configName = "config." + strings.Replace(idc, "\n", "", -1) + ".yml"
			}
		}

		// get config
		c = &Config{}
		configor.Load(c, configName)
		c.CreatorMap = make(map[string]*Creator)
		if b, info := checkParam(c); !b {
			panic(info)
		}
		for _, item := range c.Creators {
			if !checkType(item.Type) {
				panic("config creator type invalid")
			}
			if !checkEmpty(item.CreatorName) {
				panic("config creator name empty")
			}
			if !checkEmpty(item.CreatorPriKey) {
				panic("config creator private key empty")
			}
			ct := &Creator{CreatorName: item.CreatorName, CreatorPriKey: item.CreatorPriKey}
			c.CreatorMap[item.Type] = ct
		}
	})
	return c
}

func checkParam(c *Config) (bool, string) {
	if 0 == len(c.RpcAddr) {
		return false, "config rpc addr empty"
	}
	if "" == c.RedisAddr {
		return false, "config redis addr empty"
	}
	if "" == c.ApiLogPath || "" == c.JobLogPath {
		return false, "config log path empty"
	}
	if 0 == len(c.Creators) {
		return false, "config creators empty"
	}
	if "" == c.ContractDeployerName {
		return false, "config contract deployer empty"
	}
	if "" == c.ContractName {
		return false, "config contract contract empty"
	}
	if "" == c.ContractCommentMethod {
		return false, "config contract contract comment method empty"
	}
	if "" == c.ContractLikeMethod {
		return false, "config contract contract like method empty"
	}
	if "" == c.ContractSignInMethod {
		return false, "config contract contract sign method empty"
	}
	if "" == c.TransferName {
		return false, "config transfer name empty"
	}
	if "" == c.TransferPriKey {
		return false, "config transfer private key empty"
	}
	return true, ""
}

func checkType(t string) bool {
	if t != define.PGStr && t != define.ContentosStr && t != define.Game2048Str {
		return false
	}
	return true
}

func checkEmpty(s string) bool {
	if "" == s {
		return false
	}
	return true
}
