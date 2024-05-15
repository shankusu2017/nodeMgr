package main

import (
	"fmt"
	"github.com/shankusu2017/proto_pb/go/proto"
	"github.com/shankusu2017/utils"
	"math/rand"
	"testing"
	"time"
)

func TestDBInit(t *testing.T) {
	InitDB("./etc/nodeInfo.db")
	//FindNetConfigItemByUuid("ff")
}

func TestDBInsert(t *testing.T) {
	InitDB("./etc/nodeInfo.db")

	subId := int(rand.Uint32()%255) + 100000
	node := &NodeT{
		Uuid:     utils.MakeHexString(4),
		IP:       fmt.Sprintf("192.168.1.%d", subId),
		SubId:    subId,
		RoleType: 101,
		Ping:     time.Now(),
		Ver:      "ver-test",
	}
	err := InsertNetConfig(node)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// 唯一性检查
	subId = int(rand.Uint32()%255) + 200000
	node = &NodeT{
		Uuid:     utils.MakeHexString(4),
		IP:       fmt.Sprintf("192.168.1.%d", subId),
		SubId:    subId,
		RoleType: 101,
		Ping:     time.Now(),
		Ver:      "ver-test",
	}
	err = InsertNetConfig(node)
	if err != nil {
		t.Fatalf(err.Error())
	}
	err = InsertNetConfig(node)
	if err == nil {
		t.Fatalf(err.Error())
	}
}

func TestUpdateNetConfigRowByUuid(t *testing.T) {
	InitDB("./etc/nodeInfo.db")

	subId := int(rand.Uint32()%255) + 100000
	node := &NodeT{
		Uuid:     utils.MakeHexString(4),
		IP:       fmt.Sprintf("192.168.1.%d", subId),
		SubId:    subId,
		RoleType: 101,
		Ping:     time.Now(),
		Ver:      "ver-test-insert",
	}

	InsertNetConfig(node)

	node.Ver = "ver-test-update"
	UpdateNetConfigRowByUuid(node)

	retLst, _ := FindNetConfigItemByUuid(node.Uuid)
	if len(retLst) != 1 {
		t.Fatalf("0x15734373 db error")
	}

	dbNode := retLst[0]
	if dbNode.Uuid != node.Uuid ||
		dbNode.IP != node.IP ||
		dbNode.SubId != node.SubId ||
		dbNode.RoleType != node.RoleType ||
		dbNode.TS.Unix() != node.Ping.Unix() ||
		dbNode.Ver != node.Ver {
		t.Fatalf("0x7ddb47fa db error")
	}
}

func TestDeleteNetConfigItemByUuid(t *testing.T) {
	InitDB("./etc/nodeInfo.db")

	subId := int(rand.Uint32()%255) + 100000
	node := &NodeT{
		Uuid:     utils.MakeHexString(4),
		IP:       fmt.Sprintf("192.168.1.%d", subId),
		SubId:    subId,
		RoleType: 101,
		Ping:     time.Now(),
		Ver:      "ver-test-insert",
	}

	InsertNetConfig(node)
	retLst, _ := FindNetConfigItemByUuid(node.Uuid)
	if len(retLst) != 1 {
		t.Fatalf("0x2d348d02 db error")
	}

	DeleteNetConfigItemByUuid(node.Uuid)
	retLst, _ = FindNetConfigItemByUuid(node.Uuid)
	if len(retLst) != 0 {
		t.Fatalf("0x66b0f5a9 db error")
	}
}

func TestLoadAllRow(t *testing.T) {
	InitDB("./etc/nodeInfo.db")

	subId := int(rand.Uint32()%255) + 100000
	node := &NodeT{
		Uuid:     utils.MakeHexString(4),
		IP:       fmt.Sprintf("192.168.1.%d", subId),
		SubId:    subId,
		RoleType: 101,
		Ping:     time.Now(),
		Ver:      "ver-test-insert",
	}
	InsertNetConfig(node)

	// 刚插进入的，肯定在List中
	found := false
	allRow, _ := LoadNetConfigItemAll()
	for _, dNode := range allRow {
		if dNode.Uuid == node.Uuid {
			found = true
			break
		}
	}
	if found != true {
		t.Fatalf("db error")
	}
}

func TestInsertEvent(t *testing.T) {
	InitDB("./etc/nodeInfo.db")

	event := &proto.MsgEventPost{
		Event: proto.Event_STARTED,
		Ts:    time.Now().Unix(),
	}
	event.Machine = &proto.Machine{
		UUID: utils.MakeHexString(4),
	}
	event.Node = &proto.Node{
		Ver:  "ver-test-event-db",
		Role: 103,
	}
	InsertNodeEvent("192.168.1.1033", event)
}
