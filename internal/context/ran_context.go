package context

import (
	"net"
	"sync"
)

type RANContext struct {
	mu sync.RWMutex

	RanNodeId       string
	RanNodeName     string
	GlobalRanNodeId *GlobalRanNodeId

	Conn     net.Conn
	SctpAddr *net.TCPAddr

	SupportedTAList []SupportedTAI

	UeContexts sync.Map

	DefaultPagingDrx string
}

type GlobalRanNodeId struct {
	PlmnId         PlmnId
	GnbId          string
	GnbIdBitLength int
	N3IwfId        string
	NgEnbId        string
}

type SupportedTAI struct {
	Tai               Tai
	BroadcastPlmnList []PlmnId
}

func NewRANContext(ranNodeId string, conn net.Conn) *RANContext {
	return &RANContext{
		RanNodeId:       ranNodeId,
		Conn:            conn,
		SupportedTAList: make([]SupportedTAI, 0),
	}
}

func (ran *RANContext) AddUE(ranUeNgapId int64, ue *UEContext) {
	ran.UeContexts.Store(ranUeNgapId, ue)
}

func (ran *RANContext) RemoveUE(ranUeNgapId int64) {
	ran.UeContexts.Delete(ranUeNgapId)
}

func (ran *RANContext) GetUE(ranUeNgapId int64) (*UEContext, bool) {
	value, ok := ran.UeContexts.Load(ranUeNgapId)
	if !ok {
		return nil, false
	}
	return value.(*UEContext), true
}

func (ran *RANContext) RangeUEs(f func(ranUeNgapId int64, ue *UEContext) bool) {
	ran.UeContexts.Range(func(key, value interface{}) bool {
		ranUeNgapId := key.(int64)
		ue := value.(*UEContext)
		return f(ranUeNgapId, ue)
	})
}

func (ran *RANContext) ClearAllUEs() {
	ran.UeContexts = sync.Map{}
}
