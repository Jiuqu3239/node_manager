package main

import (
	"log"
	"node_manager/comm"
	"node_manager/conf"
	"os"
	"time"
)

func main() {
	if err := conf.InitConfig(conf.DefaultConfigPath); err != nil {
		log.Println("init config error", err)
		os.Exit(1)
	}
	config := conf.GetConfig()
	done := make(chan struct{})
	go func() {
		if err := comm.MsgServer(); err != nil {
			log.Println("run node server error", err)
			done <- struct{}{}
		}
	}()
	go func() {
		ticker := time.NewTicker(comm.ROUTE_TAB_FLASH_TIME)
		for range ticker.C {
			msg := comm.Message{
				Type: comm.MSG_HB,
				Data: comm.GetSelf(),
			}
			if err := comm.Broadcast(msg); err != nil {
				log.Println("broadcast failed", err)
			}
		}
	}()
	if config.Neighbor != "" {
		routeTabReq := comm.Message{
			Type: comm.MSG_REQ,
			Data: config.PeerAddr + ":" + config.NodePort,
		}
		if err := comm.SendMsg(config.Neighbor, routeTabReq); err != nil {
			log.Println("sync route table error", err)
			os.Exit(3)
		}
	}
	<-done
}
