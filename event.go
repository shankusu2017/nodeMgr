package main

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/shankusu2017/proto_pb/go/proto"
	pb "google.golang.org/protobuf/proto"
	"io"
	"log"
	"net/http"
	"strconv"
)

type EventHelpT struct {
	Text string `json:Text`
}

func EventPost(c *gin.Context) {
	ip := c.RemoteIP()

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("0x2dd56b9d read request body fail:%s, ip:%s", err.Error(), ip)
		return
	}

	var msg proto.MsgEventPost
	err = pb.Unmarshal(bodyBytes, &msg)
	if err != nil {
		log.Printf("0x5a9debca Invalid request body(%v), ip:%s", bodyBytes, ip)
		return
	}

	{
		jBuf, _ := json.Marshal(msg)
		log.Printf("0x09d8bb7d recv a event:[%s], cli:%s", string(jBuf), ip)
	}

	machine := msg.GetMachine()
	if machine == nil {
		log.Printf("0x630d0ded client(ip:%s) req repeater server list, machine.id is nil", ip)
		return
	}

	event := msg.GetEvent()
	if event == proto.Event_STARTED {
		NodeBootEvent(c, &msg)
	} else if event == proto.Event_KEEPALIVE {
		NodePingEvent(c, &msg)
	} else {
		log.Printf("0x1ae4262b recv invalid event(%s)", event)
		return
	}
}

func EventGet(c *gin.Context) {
	eLst, err := SelectEventAll()
	if err != nil {
		log.Printf("0x4111d800 SelectEventAll fail: %s", err)
		return
	}

	/* 过滤出指定类型的数据 */
	typeArg := c.Query("type")
	typeInt := -1
	var tLst []*EventItemDBT
	if len(typeArg) > 0 {
		typeInt, err = strconv.Atoi(typeArg)
		if err == nil {
			typeInt = -1
		}
	}
	if typeInt != -1 {
		for _, event := range eLst {
			if event.eType == typeInt {
				tLst = append(tLst, event)
			}
		}
	} else {
		tLst = eLst
	}

	c.JSON(http.StatusOK, tLst)
}

func EventHelp(c *gin.Context) {
	textHelp := `
	OPTIONS
    	 --type
			STARTED: 0 \n
			KEEPALIVE: 1 \n
			PINGLOSTPERCENT20: 1000\n
			PINGACKNULL: 1001 \n
			CLOSED: 65535
	`

	txt := &EventHelpT{
		Text: textHelp,
	}

	c.JSON(http.StatusOK, &txt)
}
