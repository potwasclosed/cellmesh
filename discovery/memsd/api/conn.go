package memsd

import (
	"github.com/davyxu/cellmesh/discovery"
	"github.com/davyxu/cellmesh/discovery/memsd/model"
	"github.com/davyxu/cellmesh/discovery/memsd/proto"
	"github.com/davyxu/cellnet"
	"github.com/davyxu/cellnet/peer"
	"github.com/davyxu/cellnet/proc"
	"github.com/davyxu/cellnet/timer"
	"time"
)

func (self *memDiscovery) clearCache() {

	self.svcCacheGuard.Lock()
	self.svcCache = map[string][]*discovery.ServiceDesc{}
	self.svcCacheGuard.Unlock()

	self.kvCacheGuard.Lock()
	self.kvCache = map[string][]byte{}
	self.kvCacheGuard.Unlock()
}

func (self *memDiscovery) connect(addr string) {
	p := peer.NewGenericPeer("tcp.Connector", "memsd", addr, model.Queue)

	proc.BindProcessorHandler(p, "memsd.cli", func(ev cellnet.Event) {

		switch msg := ev.Message().(type) {
		case *cellnet.SessionConnected:

			self.sesGuard.Lock()
			self.ses = ev.Session()
			self.sesGuard.Unlock()
			self.clearCache()
			ev.Session().Send(&proto.AuthREQ{
				Token: self.token,
			})

			{
				loopTime := timer.NewLoop(nil, time.Second*10, func(loop *timer.Loop) {
					if loop.Running() {
						ev.Session().Send(&proto.PingMemsd{})
					}
				}, nil).Notify().Start()
				ev.Session().(cellnet.ContextSet).SetContext("loopTime", loopTime)
			}
		case *cellnet.SessionClosed:
			self.token = ""
			log.Errorf("memsd discovery lost!")

			{
				var loopTime *timer.Loop
				ok := ev.Session().(cellnet.ContextSet).FetchContext("loopTime", loopTime)
				if ok {
					loopTime.Stop()
				}
			}
		case *proto.AuthACK:

			self.token = msg.Token

			if self.initWg != nil {
				// Pull的消息还要在queue里处理，这里确认处理完成后才算初始化完成
				self.initWg.Done()
			}

			log.Infof("memsd discovery ready!")

			self.triggerNotify("ready", 0)

		case *proto.ValueChangeNotifyACK:

			if model.IsServiceKey(msg.Key) {
				self.updateSvcCache(msg.SvcName, msg.Value)
			} else {
				self.updateKVCache(msg.Key, msg.Value)
			}

		case *proto.ValueDeleteNotifyACK:

			if model.IsServiceKey(msg.Key) {
				svcid := model.GetSvcIDByServiceKey(msg.Key)
				self.deleteSvcCache(svcid, msg.SvcName)
			} else {
				self.deleteKVCache(msg.Key)
			}
		}
	})

	// noDelay
	p.(cellnet.TCPSocketOption).SetSocketBuffer(1024*1024, 1024*1024, true)

	// 断线后自动重连
	p.(cellnet.TCPConnector).SetReconnectDuration(time.Second * 5)

	p.Start()

	for {

		if p.(cellnet.PeerReadyChecker).IsReady() {
			break
		}

		time.Sleep(time.Millisecond * 500)
	}

}
