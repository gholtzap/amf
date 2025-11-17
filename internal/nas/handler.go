package nas

import (
	"github.com/gavin/amf/internal/context"
	"github.com/gavin/amf/internal/logger"
)

type Handler struct {
	amfContext *context.AMFContext
}

func NewHandler(ctx *context.AMFContext) *Handler {
	return &Handler{
		amfContext: ctx,
	}
}

func (h *Handler) HandleRegistrationRequest(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Registration Request for UE SUPI: %s", ue.Supi)

	return nil
}

func (h *Handler) HandleDeregistrationRequest(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Deregistration Request for UE SUPI: %s", ue.Supi)

	return nil
}

func (h *Handler) HandleServiceRequest(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Service Request for UE SUPI: %s", ue.Supi)

	return nil
}

func (h *Handler) HandleAuthenticationResponse(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Authentication Response for UE SUPI: %s", ue.Supi)

	return nil
}

func (h *Handler) HandleSecurityModeComplete(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Security Mode Complete for UE SUPI: %s", ue.Supi)

	return nil
}
