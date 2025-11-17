package ngap

import (
	"net"

	"github.com/gavin/amf/internal/context"
	"github.com/gavin/amf/internal/logger"
)

// Handler handles NGAP messages from gNB
type Handler struct {
	amfContext *context.AMFContext
}

// NewHandler creates a new NGAP handler
func NewHandler(ctx *context.AMFContext) *Handler {
	return &Handler{
		amfContext: ctx,
	}
}

// HandleNGSetupRequest handles NG Setup Request from gNB (TS 38.413 Section 8.7.1)
func (h *Handler) HandleNGSetupRequest(conn net.Conn, payload []byte) error {
	logger.NgapLog.Info("Handle NG Setup Request")
	// TODO: Implement NG Setup Request handling per TS 38.413
	return nil
}

// HandleInitialUEMessage handles Initial UE Message from gNB (TS 38.413 Section 8.6.1)
func (h *Handler) HandleInitialUEMessage(conn net.Conn, payload []byte) error {
	logger.NgapLog.Info("Handle Initial UE Message")
	// TODO: Implement Initial UE Message handling per TS 38.413
	return nil
}

// HandleUplinkNASTransport handles Uplink NAS Transport from gNB (TS 38.413 Section 8.6.3)
func (h *Handler) HandleUplinkNASTransport(conn net.Conn, payload []byte) error {
	logger.NgapLog.Info("Handle Uplink NAS Transport")
	// TODO: Implement Uplink NAS Transport handling per TS 38.413
	return nil
}

// HandleInitialContextSetupResponse handles Initial Context Setup Response
func (h *Handler) HandleInitialContextSetupResponse(conn net.Conn, payload []byte) error {
	logger.NgapLog.Info("Handle Initial Context Setup Response")
	// TODO: Implement per TS 38.413
	return nil
}

// HandlePDUSessionResourceSetupResponse handles PDU Session Resource Setup Response
func (h *Handler) HandlePDUSessionResourceSetupResponse(conn net.Conn, payload []byte) error {
	logger.NgapLog.Info("Handle PDU Session Resource Setup Response")
	// TODO: Implement per TS 38.413
	return nil
}

// HandleUEContextReleaseRequest handles UE Context Release Request
func (h *Handler) HandleUEContextReleaseRequest(conn net.Conn, payload []byte) error {
	logger.NgapLog.Info("Handle UE Context Release Request")
	// TODO: Implement per TS 38.413
	return nil
}

// SendDownlinkNASTransport sends Downlink NAS Transport to gNB
func (h *Handler) SendDownlinkNASTransport(ranUeNgapId, amfUeNgapId int64, payload []byte) error {
	logger.NgapLog.Info("Send Downlink NAS Transport")
	// TODO: Implement Downlink NAS Transport per TS 38.413
	return nil
}
