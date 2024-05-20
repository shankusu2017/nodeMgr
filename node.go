package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/shankusu2017/proto_pb/go/proto"
	"github.com/shankusu2017/utils"
	pb "google.golang.org/protobuf/proto"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	SUBNET_PAC_MIN      = 20
	SUBNET_PAC_MAX      = 49
	SUBNET_REPEATER_MIN = 120
	SUBNET_REPEATER_MAX = 199
)

type nodeMgrT struct {
	nodeUuidMap     map[string]*NodeT // uuid->node
	nodeSubNetIdMap map[int]*NodeT    // subNetId->node
	dataMtx         sync.Mutex
}

type NodeT struct {
	Uuid     string    `json:"uuid,omitempty"`
	IP       string    `json:"ip,omitempty"`
	SubId    int       `json:"SubId,omitempty"` // 子网 ID
	RoleType int       `json:"roleType,omitempty"`
	Ping     time.Time `json:"ping,omitempty"` // 最后一次 ping 的时间
	Ver      string    `json:"ver,omitempty"`
}

var (
	nodeMgr *nodeMgrT
)

// 更新 ping 时间戳
func (mgr *nodeMgrT) updateNode(ip, uuid string) {
	mgr.dataMtx.Lock()
	defer mgr.dataMtx.Unlock()

	node, ok := mgr.nodeUuidMap[uuid]
	if !ok {
		log.Printf("ERROR 0x5a43bf8d node is nil, uuid:%s, cli.ip: %s", uuid, ip)
		return
	}

	node.Ping = time.Now()
}

// 查找指定的 Node
func (mgr *nodeMgrT) findNode(uuid string) *NodeT {
	mgr.dataMtx.Lock()
	defer mgr.dataMtx.Unlock()

	node, _ := mgr.nodeUuidMap[uuid]
	return node
}

// 根据参数，新增一个 Node,并插入 map
func (mgr *nodeMgrT) newNode(uuid, ip, ver string) *NodeT {
	mgr.dataMtx.Lock()
	defer mgr.dataMtx.Unlock()

	var node = NodeT{}
	node.Uuid = uuid
	node.IP = ip
	subNetAllocDone := false

	localIP := utils.IsLocalIP(ip)
	if localIP {
		for i := SUBNET_PAC_MIN; i <= SUBNET_PAC_MAX; i++ {
			_, exist := mgr.nodeSubNetIdMap[i]
			if exist == false {
				node.SubId = i
				subNetAllocDone = true
				break
			}
		}
		node.RoleType = int(proto.Role_Pac)
	} else {
		for i := SUBNET_REPEATER_MIN; i <= SUBNET_REPEATER_MAX; i++ {
			_, exist := mgr.nodeSubNetIdMap[i]
			if exist == false {
				node.SubId = i
				subNetAllocDone = true
				break
			}
		}
		node.RoleType = int(proto.Role_Repeater)
	}
	if subNetAllocDone != true {
		return nil
	}

	node.Ping = time.Now()
	node.Ver = ver

	mgr.nodeUuidMap[node.Uuid] = &node
	mgr.nodeSubNetIdMap[node.SubId] = &node

	return &node
}

func (mgr *nodeMgrT) switchNodeSubNetIdRoleType(oldId int, newRole int, node *NodeT) bool {
	mgr.dataMtx.Lock()
	defer mgr.dataMtx.Unlock()

	if int(proto.Role_Pac) == newRole {
		for i := SUBNET_PAC_MIN; i <= SUBNET_PAC_MAX; i++ {
			_, exist := mgr.nodeSubNetIdMap[i]
			if exist == false {
				delete(mgr.nodeSubNetIdMap, oldId)
				node.SubId = i
				node.RoleType = newRole
				node.Ping = time.Now()
				mgr.nodeSubNetIdMap[i] = node
				return true
			}
		}
	} else {
		for i := SUBNET_REPEATER_MIN; i <= SUBNET_REPEATER_MAX; i++ {
			_, exist := mgr.nodeSubNetIdMap[i]
			if exist == false {
				delete(mgr.nodeSubNetIdMap, oldId)
				node.SubId = i
				node.RoleType = newRole
				node.Ping = time.Now()
				mgr.nodeSubNetIdMap[i] = node
				return true
			}
		}
	}

	return false
}

// 获取指定角色(pac或repeater)的node列表
func (mgr *nodeMgrT) getNodeIPByRoleType(roleType int) []string {
	mgr.dataMtx.Lock()
	defer mgr.dataMtx.Unlock()

	ipList := make([]string, 0)
	for _, node := range mgr.nodeUuidMap {
		if node.RoleType == roleType {
			ipList = append(ipList, node.IP)
		}
	}

	return ipList
}

// 删除过期的 node
func (mgr *nodeMgrT) loopScanDeadNode() {
	for {
		time.Sleep(time.Minute * 1)
		now := time.Now()

		mgr.dataMtx.Lock()
		for uuid, node := range mgr.nodeUuidMap {
			// 有效期N分钟，后期可以再根据实际情况优化
			if node.Ping.Before(now.Add(time.Minute * -30)) {
				log.Printf("LOG 0x71bec216 node.uuid(%s) too old, delete it", uuid)
				delete(mgr.nodeUuidMap, uuid)
				delete(mgr.nodeSubNetIdMap, node.SubId)
				err := DeleteNetConfigItemByUuid(uuid)
				if err != nil {
					log.Printf("%s", err)
				}
			}
		}
		mgr.dataMtx.Unlock()
	}
}

func NodeBootEvent(c *gin.Context, msg *proto.MsgEventPost) {
	ip := c.RemoteIP()

	var node *NodeT
	mMachine := msg.GetMachine()
	if mMachine == nil {
		log.Printf("ERROR 0x443ff8d3 machine is nil, cli.ip:%s", ip)
		return
	}
	mNode := msg.GetNode()
	if mNode == nil {
		log.Printf("ERROR 0x12912233 newNode is nil")
		return
	}
	uuid := mMachine.GetUUID()
	ver := mNode.GetVer()
	isNewNode := false
	addMsg := ""

	// 新生成的 Node 还是已有的 Node?
	node = nodeMgr.findNode(uuid)
	if node == nil {
		node = nodeMgr.newNode(uuid, ip, ver)
		if node == nil {
			log.Printf("ERROR 0x554a57ea newNode fail, uuid:%s", uuid)
			return
		}
		isNewNode = true
	} else {
		isLocal := utils.IsLocalIP(ip)
		// 角色没变，沿用之前的子网参数
		if (isLocal && node.RoleType == int(proto.Role_Pac)) || (isLocal == false && node.RoleType == int(proto.Role_Repeater)) {
			// node.RoleType = proto.Role_Pac
		} else {
			newRole := proto.Role_Default
			if isLocal {
				newRole = proto.Role_Pac
			} else {
				newRole = proto.Role_Repeater
			}
			// 尝试切换到新的角色并获取新的网络参数，释放旧的参数
			done := nodeMgr.switchNodeSubNetIdRoleType(node.SubId, int(newRole), node)
			if done == false {
				log.Printf("ERROR 0x74b3cf20 switchNodeSubNetIdRoleType fail")
				return
			}
			addMsg = fmt.Sprintf("switch 2 newType: %d, subNet: %d", node.RoleType, node.SubId)
		}
	}

	// 刷新下
	node.IP = ip
	node.Ping = time.Now()
	node.Ver = ver

	if isNewNode {
		InsertNetConfig(node)
	} else {
		UpdateNetConfigRowByUuid(node)
	}

	addMsg = fmt.Sprintf("%s roleType.now: %d", addMsg, node.RoleType)
	eMsg := msg.GetMsg()
	if eMsg != nil {
		addMsg = fmt.Sprintf("%s [%s]", eMsg.Msg, addMsg)
	}

	// 存DB
	InsertNodeEvent(ip, proto.Role(node.RoleType), addMsg, msg)

	{
		var rsp proto.MsgEventRsp
		rsp.Event = msg.Event
		rsp.Machine = &proto.Machine{
			UUID: uuid,
		}
		rsp.Node = &proto.Node{
			Role: proto.Role(node.RoleType),
		}
		rsp.Net = &proto.Net{
			SubId: int32(node.SubId),
		}
		// 返回网络参数给 node
		c.ProtoBuf(http.StatusOK, &rsp)
	}
}

func NodePingEvent(c *gin.Context, msg *proto.MsgEventPost) {
	ip := c.RemoteIP()
	mMachine := msg.GetMachine()
	if mMachine == nil {
		log.Printf("ERROR 0x19baf3a4 uuid is nil, cli.ip:%s", c.RemoteIP())
		return
	}
	uuid := msg.GetMachine().GetUUID()

	nodeMgr.updateNode(ip, uuid)

	err := UpdateNetConfigPingByUuid(uuid)
	if err != nil {
		log.Printf("0x56d90c0b ping update err:%s", err)
	}
}

func NodeAbnormalEvent(c *gin.Context, msg *proto.MsgEventPost) {
	// 存DB
	if msg.GetNode() == nil || msg.GetMachine() == nil || msg.GetMsg() == nil {
		jsonTxt, _ := json.Marshal(msg)
		log.Printf("ERROR 0x7e07ffea data has nil, cli.ip:%s, packet.json:%s", c.RemoteIP(), string(jsonTxt))
		return
	}
	InsertNodeEvent(c.RemoteIP(), proto.Role(msg.GetNode().Role), msg.GetMsg().Msg, msg)
}

func NodeMgrInit(dbPath string) {
	nodeMgr = &nodeMgrT{}
	nodeMgr.nodeUuidMap = make(map[string]*NodeT)
	nodeMgr.nodeSubNetIdMap = make(map[int]*NodeT)

	InitDB(dbPath)
	allNode, err := LoadNetConfigItemAll()
	if err != nil {
		log.Fatal(err)
	}

	/* 加载、校验数据 */
	for _, node := range allNode {
		var n NodeT
		n.Uuid = node.Uuid
		n.IP = node.IP
		n.SubId = node.SubId
		n.RoleType = node.RoleType
		n.Ping = node.TS
		n.Ver = node.Ver
		{ // 不得重复
			_, existA := nodeMgr.nodeUuidMap[node.Uuid]
			_, existB := nodeMgr.nodeSubNetIdMap[node.SubId]
			if existA == true || existB == true {
				log.Fatal(fmt.Sprintf("0x5b6cd7cc uuid or subId exist, uuid: %s, subNetId:%d", node.Uuid, node.SubId))
			}
		}
		nodeMgr.nodeUuidMap[node.Uuid] = &n
		nodeMgr.nodeSubNetIdMap[node.SubId] = &n
	}

	go nodeMgr.loopScanDeadNode()
}

func NodeGetAll() []NodeT {
	nodeMgr.dataMtx.Lock()
	defer nodeMgr.dataMtx.Unlock()

	lst := make([]NodeT, 0)

	for _, node := range nodeMgr.nodeUuidMap {
		lst = append(lst, *node)
	}

	return lst
}

func NodeRepeaterGet(c *gin.Context) {
	ip := c.RemoteIP()

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("0x66dda745 read request body fail:%s, ip:%s", err.Error(), ip)
		return
	}

	var req proto.MsgRepeaterServerInfoReq
	err = pb.Unmarshal(bodyBytes, &req)
	if err != nil {
		log.Printf("0x4842fc43 Invalid request body(%v), ip:%s", bodyBytes, ip)
		return
	}

	machine := req.GetMachine()
	if machine == nil {
		log.Printf("0x630d0ded client(ip:%s) req repeater server list, machine.id is nil", ip)
		return
	}
	log.Printf("0x2e6b9922 req repeater server list client(ip:%s, id:%s)", ip, machine.GetUUID())

	var rsp proto.MsgRepeaterServerInfoRsp

	ipLst := nodeMgr.getNodeIPByRoleType(int(proto.Role_Repeater))
	for _, iP := range ipLst {
		node := &proto.RepeaterServerNode{IPv4: iP}
		rsp.Servers = append(rsp.Servers, node)
	}
	log.Printf("DEBUG 0x2eda1c94 iplst:%v", ipLst)

	c.ProtoBuf(http.StatusOK, &rsp)
}
