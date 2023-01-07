package comm

import (
	"encoding/json"
	"log"
	"net"
	"node_manager/conf"
	"node_manager/manager"
	"node_manager/utils"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	MSG_DATA = "data"
	MSG_HB   = "heartbeat"
	MSG_REQ  = "request"
)

const (
	ROUTE_TAB_PATH       = "./comm/route.json"
	ROUTE_TAB_FLASH_TIME = 10 * time.Second
)

type RouteItem struct {
	PeerID    string
	GroupID   string
	PeerAddr  string
	PeerPort  string
	NodePort  string
	FlashTime time.Time
}

type Message struct {
	Type string
	Data any
	Sign []byte
}

type RouteTable struct {
	mu  sync.Mutex
	tab []RouteItem
	alt bool
}

func (t *RouteTable) Update(item RouteItem) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := 0; i < len(t.tab); i++ {
		if item.PeerID == t.tab[i].PeerID {
			t.tab[i] = item
			return
		}
	}
	t.alt = true
	t.tab = append(t.tab, item)
}

func (t *RouteTable) Delete(item RouteItem) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := 0; i < len(t.tab); i++ {
		if item.PeerID == t.tab[i].PeerID {
			t.tab = append(t.tab[:i], t.tab[i+1:]...)
			t.alt = true
			return
		}
	}
}

func (t *RouteTable) IsUpdate() bool {
	t.mu.Lock()
	alt := t.alt
	t.mu.Unlock()
	return alt
}

func (t *RouteTable) FlashTable() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.alt = false
	ctab := make([]RouteItem, len(t.tab))
	copy(ctab, t.tab)
	return SaveRouteTable(ctab)
}

func (t *RouteTable) GetTable() []RouteItem {
	t.mu.Lock()
	res := make([]RouteItem, len(t.tab))
	copy(res, t.tab)
	t.mu.Unlock()
	return res
}

func Broadcast(msg Message) error {
	routeTab, err := LoadRouteTable()
	if err != nil {
		return errors.Wrap(err, "load route table error")
	}
	var wg sync.WaitGroup
	for _, item := range routeTab {
		wg.Add(1)
		url := item.PeerAddr + ":" + item.NodePort
		go func() {
			defer wg.Done()
			if err := SendMsg(url, msg); err != nil {
				log.Printf("send msg to %s error:%v.\n", url, err)
			}
		}()
	}
	wg.Wait()
	return nil
}

func SendMsg(url string, msg Message) error {
	conn, err := net.Dial("tcp", url)
	if err != nil {
		return err
	}
	defer conn.Close()
	jbytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = conn.Write(jbytes)
	return err
}

func TCPServer(msgCh chan<- Message) error {
	listen, err := net.Listen("tcp", ":"+conf.GetConfig().NodePort)
	log.Println("run tcp server:", conf.GetConfig().NodePort)
	if err != nil {
		close(msgCh)
		return errors.Wrap(err, "run tcp server error")
	}
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Println("accept failed", err)
			continue
		}
		go func() {
			defer conn.Close()
			var bytesMsg []byte
			for {
				buf := make([]byte, 1024)
				n, err := conn.Read(buf)
				if err != nil {
					log.Println("read from remote node error", err)
					return
				}
				bytesMsg = append(bytesMsg, buf[:n]...)
				if n < 1024 {
					break
				}
			}
			var msg Message
			err := json.Unmarshal(bytesMsg, &msg)
			if err != nil {
				log.Println("unmarshal message error", err)
				return
			}
			msgCh <- msg
		}()
	}
}

func MsgServer() error {
	tab, err := LoadRouteTable()
	if err != nil {
		return errors.Wrap(err, "load route table error")
	}
	routeTab := &RouteTable{
		tab: tab,
	}
	go updateRouteTab(routeTab)
	msgBuf := make(chan Message, len(tab))
	go func() {
		err = TCPServer(msgBuf)
	}()
	for msg := range msgBuf {
		switch msg.Type {
		case MSG_DATA:
			jbytes, err := json.Marshal(msg.Data)
			if err != nil {
				log.Println("marshal msg data error")
				continue
			}
			var tab []RouteItem
			err = json.Unmarshal(jbytes, &tab)
			if err != nil {
				log.Println("bad response data")
			}
			for _, item := range tab {
				routeTab.Update(item)
			}
		case MSG_REQ:
			tab := routeTab.GetTable()
			tab = append(tab, GetSelf())
			resp := Message{
				Type: MSG_DATA,
				Data: tab,
			}
			url, ok := msg.Data.(string)
			if !ok {
				log.Println("bad request")
				continue
			}
			go SendMsg(url, resp)
		case MSG_HB:
			item, ok := msg.Data.(RouteItem)
			if !ok || item.PeerID == "" || item.PeerAddr == "" || item.PeerPort == "" {
				log.Println("prase message to route item error")
				continue
			}
			item.FlashTime = time.Now()
			routeTab.Update(item)
		}
	}
	return err
}

func updateRouteTab(rt *RouteTable) {
	ticker := time.NewTicker(ROUTE_TAB_FLASH_TIME)
	for t := range ticker.C {
		tab := rt.GetTable()
		var addrList []string
		for _, item := range tab {
			if t.Sub(item.FlashTime) > ROUTE_TAB_FLASH_TIME*3 {
				rt.Delete(item)
				continue
			}
			addr := item.PeerAddr + ":" + item.PeerPort
			addrList = append(addrList, addr)
		}
		if !rt.IsUpdate() {
			continue
		}
		if err := rt.FlashTable(); err != nil {
			log.Println(err)
		}
		if err := manager.UpdateNodeConf("peers", addrList); err != nil {
			log.Println(err)
			continue
		}
		if err := manager.FlashNodeConf(); err != nil {
			log.Println(err)
		}
	}
}

func LoadRouteTable() ([]RouteItem, error) {
	var routeTab []RouteItem
	jbytes, err := utils.ReadFromFile(ROUTE_TAB_PATH)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(jbytes, &routeTab)
	return routeTab, err
}

func SaveRouteTable(routeTab []RouteItem) error {
	jbytes, err := json.Marshal(routeTab)
	if err != nil {
		return err
	}
	return utils.WriteToFile(ROUTE_TAB_PATH, jbytes)
}

func GetSelf() RouteItem {
	config := conf.GetConfig()
	return RouteItem{
		PeerID:   config.PeerID,
		GroupID:  config.Group,
		PeerAddr: config.PeerAddr,
		PeerPort: config.PeerPort,
		NodePort: config.NodePort,
	}
}
