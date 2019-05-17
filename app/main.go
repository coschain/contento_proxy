package main

import (
	"proxy/database"
	"os"
	"os/signal"
	"proxy/server"
	"syscall"
	"proxy/config"
)

func main() {

	conf := config.GetConfig()

	db := database.NewDB()
	if db == nil {
		panic("init db error")
	}

	apiLogFile, _ := os.OpenFile(conf.ApiLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	jobLogFile, _ := os.OpenFile(conf.JobLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err := server.Init(db,apiLogFile,jobLogFile); err != nil {
		panic(err)
	}
	defer server.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		s := <-c
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			return
		case syscall.SIGHUP:
		default:
			return
		}
	}
}
