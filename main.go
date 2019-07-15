package main

import (
	"fmt"
	go_wechat "github.com/oliverCJ/go-wechat"
	layout2 "github.com/oliverCJ/wechat-terminal/layout"
	"os"
)

func main() {
	dir, _ := os.Getwd()
	go_wechat.SetHoReload(true)
	go_wechat.SetRootPath(dir)

	logOutChan := make(chan string, 100)
	logOutFile, _ := os.OpenFile(os.DevNull, os.O_RDWR, 0644)
	defer logOutFile.Close()
	go_wechat.SetLog("warn", logOutChan, logOutFile)
	logRecordFile, err := os.OpenFile(dir + "/wxlog.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	defer logRecordFile.Close()
	if err != nil {
		fmt.Print("打开日志文件失败")
		return
	}

	go func() {

		for {
			select {
			case logMsg := <- logOutChan:
				fmt.Fprint(logRecordFile, logMsg)
			}
		}
	}()

	if err := go_wechat.Start(); err != nil {
		return
	}
	layout := layout2.NewLayout()
	layout.Init()
}
