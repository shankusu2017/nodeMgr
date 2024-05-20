package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/shankusu2017/proto_pb/go/proto"
	"log"
	"time"
)

var (
	dbHandle *sql.DB
)

// NetConfigT 网络参数
type NetConfigT struct {
	SubId    int       `json:SubId`
	Uuid     string    `json:Uuid`
	IP       string    `json:IP`
	RoleType int       `json:RoleType`
	TS       time.Time `json:TS`
	Ver      string    `json:Ver`
}

type EventItemDBT struct {
	Uuid     string    `json:Uuid`
	IP       string    `json:IP`
	RoleType int       `json:RoleType`
	TS       time.Time `json:TS`
	Ver      string    `json:Ver`
	eType    int       `json:eType`
	eMsg     string    `json:eMsg`
}

func openDB(dbPath string) *sql.DB {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	//db.Close()
	return db
}

func createTable(db *sql.DB) error {
	// 10.x.0.0 x 这个子网号的分配
	// sub_id 子网号
	// uuid 设备标识符
	// ip node 的公网 ip
	// roleType 角色类型
	// ver 版本号
	// ts 分配时间
	sqlStmt := `
	create table IF NOT EXISTS netConfigTbl (
		sub_id INT NOT NULL PRIMARY KEY,
	    uuid text unique,
	    ip text,
	    roleType INT,
	    ver text,
	    ts timestamp);
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%s: %s\n", err.Error(), sqlStmt)
		return err
	}

	// 事件记录表
	// id 序列号
	// uuid
	// ip
	// roleType
	// ver
	// eventType
	// eventMsg
	// ts
	sqlStmt = `
	create table IF NOT EXISTS nodeEventTbl (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid text,
		ip text,
		roleType INT,
		ver text,
		eventType INT,
		eventMsg text,
		ts timestamp);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%s: %s\n", err.Error(), sqlStmt)
		return err
	}

	return nil
}

func InitDB(dbPath string) {
	dbHandle = openDB(dbPath)
	err := createTable(dbHandle)
	if err != nil {
		log.Fatal(err)
	}
}

func LoadNetConfigItemAll() ([]*NetConfigT, error) {
	var retLst []*NetConfigT

	rows, err := dbHandle.Query("SELECT sub_id, uuid, ip, roleType, ver, ts FROM netConfigTbl")
	if err != nil {
		log.Printf("0x6625c105 db.Query err:%s", err)
		return retLst, err
	}
	defer rows.Close()

	for rows.Next() {
		var node NetConfigT
		err = rows.Scan(&node.SubId, &node.Uuid, &node.IP, &node.RoleType, &node.Ver, &node.TS)
		if err != nil {
			log.Printf("0x50f73e51 rows.Scan err:%s", err)
			return nil, err
		}
		//buf, _ := json.Marshal(&node)
		//log.Printf("jons:[%s]", string(buf))
		retLst = append(retLst, &node)
	}
	err = rows.Err()
	if err != nil {
		log.Printf("0x2b122202 rows err:%s", err)
		return []*NetConfigT{}, err
	}

	return retLst, nil
}

// FindNetConfigItemByUuid 不保证结果唯一
func FindNetConfigItemByUuid(uuid string) ([]*NetConfigT, error) {
	var retLst []*NetConfigT

	rows, err := dbHandle.Query("SELECT sub_id, ip, roleType, ver, ts FROM netConfigTbl WHERE uuid == ?", uuid)
	if err != nil {
		log.Printf("0x255d7d91 db.Query err:%s", err)
		return retLst, err
	}
	defer rows.Close()

	for rows.Next() {
		var node NetConfigT
		err = rows.Scan(&node.SubId, &node.IP, &node.RoleType, &node.Ver, &node.TS)
		if err != nil {
			log.Printf("0x3c8d5220 rows.Scan err:%s", err)
			return nil, err
		}
		node.Uuid = uuid
		//buf, _ := json.Marshal(&node)
		//log.Printf("jons:[%s]", string(buf))
		retLst = append(retLst, &node)
	}
	err = rows.Err()
	if err != nil {
		log.Printf("0x2b122202 rows err:%s", err)
		return []*NetConfigT{}, err
	}

	return retLst, nil
}

// DeleteNetConfigItemByUuid 删除指定行
func DeleteNetConfigItemByUuid(uuid string) error {
	// 删除数据
	deleteStmt, err := dbHandle.Prepare("DELETE FROM netConfigTbl WHERE uuid = ?")
	if err != nil {
		return errors.New(fmt.Sprintf("0x3ca6aadb db.Prepare error uuid:%s", uuid))
	}
	defer deleteStmt.Close()

	_, err = deleteStmt.Exec(uuid)
	if err != nil {
		return errors.New(fmt.Sprintf("0x0faf25ad stmt.Exec error uuid:%s", uuid))
	}

	return nil
}

func UpdateNetConfigPingByUuid(uuid string) error {
	updateStmt, err := dbHandle.Prepare("UPDATE netConfigTbl SET ts = ? WHERE uuid = ?")
	if err != nil {
		return errors.New(fmt.Sprintf("0x0454bd4e db.Prepare error uuid:%s", uuid))
	}
	defer updateStmt.Close()

	_, err = updateStmt.Exec(time.Now(), uuid)
	if err != nil {
		return errors.New(fmt.Sprintf("0x60fd39d8 stmt.Exec error uuid:%s", uuid))
	}

	return nil
}

// UpdateNetConfigRowByUuid 更新原有数据
func UpdateNetConfigRowByUuid(node *NodeT) error {
	ctx := context.Background()
	rowInfo := fmt.Sprintf("subId:%d, uuid:%s, ip:%s, roleType:%d, ver:%s, ts:%s", node.SubId, node.Uuid, node.IP, node.RoleType, node.Ver, node.Ping)

	result, err := dbHandle.ExecContext(ctx, "UPDATE netConfigTbl SET sub_id=?, ip=?, roleType=?, ver=?, ts=? WHERE uuid=?", node.SubId, node.IP, node.RoleType, node.Ver, node.Ping, node.Uuid)
	if err != nil {
		return errors.New(fmt.Sprintf("0x64d8c6a9 update fail(%s), row:%s", err, rowInfo))
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return errors.New(fmt.Sprintf("0x2ddc70c6 update fail(%s), row:%s", err, rowInfo))
	}
	if rows != 1 {
		return errors.New(fmt.Sprintf("0x6adec0c4 update fail(%s), row:%s", err, rowInfo))
	}

	return nil
}

// InsertNetConfig 新增一条数据
func InsertNetConfig(node *NodeT) error {
	rowInfo := fmt.Sprintf("row info subId:%d, uuid:%s, ip:%s, role:%d, ver:%s ts:%v",
		node.SubId, node.Uuid, node.IP, node.RoleType, node.Ver, node.Ping)

	stmt, err := dbHandle.Prepare("INSERT INTO netConfigTbl(sub_id, uuid, ip, roleType, ver, ts) VALUES ( ?, ?, ?, ?, ?, ? )")
	if err != nil {
		err = errors.New(fmt.Sprintf("0x159f4891 db.Prepare fail:%s, rowInfo:%s", err, rowInfo))
		log.Printf(err.Error())
		return err
	}
	defer stmt.Close() // Prepared statements take up server resources and should be closed after use.

	_, err = stmt.Exec(node.SubId, node.Uuid, node.IP, node.RoleType, node.Ver, node.Ping)
	if err != nil {
		err = errors.New(fmt.Sprintf("0x777e5d3e insert fail:%s, rowInfo:%v", err, rowInfo))
		log.Printf(err.Error())
		return err
	}

	return nil
}

// InsertNodeEvent 新增一条数据
func InsertNodeEvent(ip string, role proto.Role, addMsg string, event *proto.MsgEventPost) error {
	rowInfo, _ := json.Marshal(*event)
	//	create table IF NOT EXISTS nodeEventTbl (id INT NOT NULL AUTO_INCREMENT PRIMARY KEY, uuid text, ip text, ver text, eventType INT, eventMsg text, roleType INT, ts timestamp);
	stmt, err := dbHandle.Prepare("INSERT INTO nodeEventTbl(uuid, ip, roleType, ver, eventType, eventMsg, ts) VALUES ( ?, ?, ?, ?, ?, ?, ? )")
	if err != nil {
		err = errors.New(fmt.Sprintf("0x159f4891 db.Prepare fail:%s, rowInfo:%v", err, string(rowInfo)))
		log.Printf(err.Error())
		return err
	}
	defer stmt.Close() // Prepared statements take up server resources and should be closed after use.

	_, err = stmt.Exec(event.Machine.GetUUID(), ip, role, event.Node.GetVer(), event.Event, addMsg, time.Now())
	if err != nil {
		err = errors.New(fmt.Sprintf("0x2bda2151 insert fail:%s, %v", err, rowInfo))
		log.Printf(err.Error())
		return err
	}

	return nil
}

func SelectEventAll() ([]*EventItemDBT, error) {
	var retLst []*EventItemDBT

	rows, err := dbHandle.Query("SELECT uuid, ip, roleType, ver, eventType, eventMsg, ts FROM nodeEventTbl")
	if err != nil {
		log.Printf("0x645df775 db.Query err:%s", err)
		return retLst, err
	}
	defer rows.Close()

	for rows.Next() {
		event := &EventItemDBT{}
		err = rows.Scan(&event.Uuid, &event.IP, &event.RoleType, &event.Ver, &event.eType, &event.eMsg, &event.TS)
		if err != nil {
			log.Printf("0x28f973da rows.Scan err:%s", err)
			return nil, err
		}
		//buf, _ := json.Marshal(&node)
		//log.Printf("jons:[%s]", string(buf))
		retLst = append(retLst, event)
	}
	err = rows.Err()
	if err != nil {
		log.Printf("0x144d7208 rows err:%s", err)
		return []*EventItemDBT{}, err
	}

	return retLst, nil
}
