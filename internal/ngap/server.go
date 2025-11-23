package ngap

import (
	"fmt"
	"net"

	"github.com/gavin/amf/internal/context"
	"github.com/gavin/amf/internal/logger"
	"github.com/ishidawataru/sctp"
)

type Server struct {
	handler    *Handler
	listener   *sctp.SCTPListener
	amfContext *context.AMFContext
}

func NewServer(amfCtx *context.AMFContext, handler *Handler) *Server {
	return &Server{
		handler:    handler,
		amfContext: amfCtx,
	}
}

func (s *Server) Listen(addr string, port int) error {
	laddr, err := sctp.ResolveSCTPAddr("sctp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return fmt.Errorf("failed to resolve SCTP address: %w", err)
	}

	listener, err := sctp.ListenSCTP("sctp", laddr)
	if err != nil {
		return fmt.Errorf("failed to listen on SCTP: %w", err)
	}

	s.listener = listener
	logger.NgapLog.Infof("NGAP server listening on %s:%d", addr, port)

	return nil
}

func (s *Server) Serve() error {
	if s.listener == nil {
		return fmt.Errorf("server not initialized, call Listen first")
	}

	for {
		conn, err := s.listener.AcceptSCTP()
		if err != nil {
			logger.NgapLog.Errorf("Failed to accept connection: %v", err)
			continue
		}

		logger.NgapLog.Infof("New SCTP connection from %s", conn.RemoteAddr())
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn *sctp.SCTPConn) {
	defer conn.Close()

	ranContext := context.NewRANContext("", conn)
	s.amfContext.RanContexts.Store(conn.RemoteAddr().String(), ranContext)
	defer s.amfContext.RanContexts.Delete(conn.RemoteAddr().String())

	buffer := make([]byte, 65535)
	for {
		n, _, err := conn.SCTPRead(buffer)
		if err != nil {
			logger.NgapLog.Infof("Connection closed: %v", err)
			return
		}

		if n < 1 {
			continue
		}

		if err := s.handleMessage(ranContext, buffer[:n]); err != nil {
			logger.NgapLog.Errorf("Failed to handle message: %v", err)
		}
	}
}

func (s *Server) handleMessage(ranContext *context.RANContext, data []byte) error {
	if len(data) < 1 {
		return fmt.Errorf("message too short")
	}

	pdu, err := DecodeNGAPPDU(data)
	if err != nil {
		return fmt.Errorf("failed to decode NGAP PDU: %w", err)
	}

	logger.NgapLog.Infof("Received NGAP message: procedure code %d, criticality %d, type %s",
		pdu.ProcedureCode, pdu.Criticality, pdu.Type)

	switch pdu.ProcedureCode {
	case ProcedureCodeNGSetup:
		return s.handler.HandleNGSetupRequest(ranContext, pdu)
	case ProcedureCodeInitialUEMessage:
		return s.handler.HandleInitialUEMessage(ranContext, pdu)
	case ProcedureCodeUplinkNASTransport:
		return s.handler.HandleUplinkNASTransport(ranContext, pdu)
	case ProcedureCodeInitialContextSetup:
		return s.handler.HandleInitialContextSetupResponse(ranContext, pdu)
	case ProcedureCodePDUSessionResourceSetup:
		return s.handler.HandlePDUSessionResourceSetupResponse(ranContext, pdu)
	case ProcedureCodeUEContextRelease:
		return s.handler.HandleUEContextReleaseRequest(ranContext, pdu)
	case ProcedureCodePathSwitchRequest:
		return s.handler.HandlePathSwitchRequest(ranContext, pdu)
	case ProcedureCodeNGReset:
		return s.handler.HandleNGReset(ranContext, pdu)
	case ProcedureCodeErrorIndication:
		return s.handler.HandleErrorIndication(ranContext, pdu)
	case ProcedureCodeOverloadStart:
		return s.handler.HandleOverloadStart(ranContext, pdu)
	case ProcedureCodeOverloadStop:
		return s.handler.HandleOverloadStop(ranContext, pdu)
	case ProcedureCodeAMFConfigurationUpdate:
		if pdu.Type == PDUTypeSuccessfulOutcome {
			return s.handler.HandleAMFConfigurationUpdateAcknowledge(ranContext, pdu)
		} else if pdu.Type == PDUTypeUnsuccessfulOutcome {
			return s.handler.HandleAMFConfigurationUpdateFailure(ranContext, pdu)
		}
		logger.NgapLog.Warnf("Unexpected PDU type for AMF Configuration Update: %s", pdu.Type)
		return nil
	case ProcedureCodeRANConfigurationUpdate:
		return s.handler.HandleRANConfigurationUpdate(ranContext, pdu)
	case ProcedureCodeUERadioCapabilityInfoIndication:
		return s.handler.HandleUERadioCapabilityInfoIndication(ranContext, pdu)
	case ProcedureCodeLocationReport:
		return s.handler.HandleLocationReport(ranContext, pdu)
	case ProcedureCodeHandoverPreparation:
		return s.handler.HandleHandoverRequired(ranContext, pdu)
	case ProcedureCodeHandoverResourceAllocation:
		if pdu.Type == PDUTypeSuccessfulOutcome {
			return s.handler.HandleHandoverRequestAcknowledge(ranContext, pdu)
		}
		logger.NgapLog.Warnf("Unexpected PDU type for Handover Resource Allocation: %s", pdu.Type)
		return nil
	case ProcedureCodeHandoverNotification:
		return s.handler.HandleHandoverNotify(ranContext, pdu)
	case ProcedureCodeRANCPRelocationIndication:
		return s.handler.HandleRANCPRelocationIndication(ranContext, pdu)
	default:
		logger.NgapLog.Warnf("Unsupported procedure code: %d", pdu.ProcedureCode)
		return nil
	}
}

func (s *Server) SendMessage(conn net.Conn, pdu *NGAPPDU) error {
	data, err := EncodeNGAPPDU(pdu)
	if err != nil {
		return fmt.Errorf("failed to encode NGAP PDU: %w", err)
	}

	sctpConn, ok := conn.(*sctp.SCTPConn)
	if !ok {
		return fmt.Errorf("connection is not SCTP")
	}

	_, err = sctpConn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	logger.NgapLog.Infof("Sent NGAP message: procedure code %d", pdu.ProcedureCode)
	return nil
}

func (s *Server) Close() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}
