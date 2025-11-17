package metrics

import (
	"sync/atomic"
)

// Metrics represents AMF metrics
type Metrics struct {
	// Registration metrics
	RegistrationAttempts  atomic.Uint64
	RegistrationSuccesses atomic.Uint64
	RegistrationFailures  atomic.Uint64

	// Deregistration metrics
	DeregistrationAttempts  atomic.Uint64
	DeregistrationSuccesses atomic.Uint64

	// Service Request metrics
	ServiceRequestAttempts  atomic.Uint64
	ServiceRequestSuccesses atomic.Uint64
	ServiceRequestFailures  atomic.Uint64

	// Authentication metrics
	AuthenticationAttempts  atomic.Uint64
	AuthenticationSuccesses atomic.Uint64
	AuthenticationFailures  atomic.Uint64

	// PDU Session metrics
	PduSessionEstablishments atomic.Uint64
	PduSessionReleases       atomic.Uint64
	ActivePduSessions        atomic.Int64

	// UE metrics
	RegisteredUEs atomic.Int64
	ConnectedUEs  atomic.Int64

	// NGAP metrics
	NgSetupAttempts       atomic.Uint64
	NgSetupSuccesses      atomic.Uint64
	InitialUeMessages     atomic.Uint64
	UplinkNasTransports   atomic.Uint64
	DownlinkNasTransports atomic.Uint64

	// Error metrics
	NasDecodeErrors  atomic.Uint64
	NgapDecodeErrors atomic.Uint64
}

var amfMetrics *Metrics

func init() {
	amfMetrics = &Metrics{}
}

// GetMetrics returns the AMF metrics instance
func GetMetrics() *Metrics {
	return amfMetrics
}

// IncrementRegistrationAttempts increments registration attempts
func (m *Metrics) IncrementRegistrationAttempts() {
	m.RegistrationAttempts.Add(1)
}

// IncrementRegistrationSuccesses increments registration successes
func (m *Metrics) IncrementRegistrationSuccesses() {
	m.RegistrationSuccesses.Add(1)
}

// IncrementRegistrationFailures increments registration failures
func (m *Metrics) IncrementRegistrationFailures() {
	m.RegistrationFailures.Add(1)
}

// IncrementAuthenticationAttempts increments authentication attempts
func (m *Metrics) IncrementAuthenticationAttempts() {
	m.AuthenticationAttempts.Add(1)
}

// IncrementAuthenticationSuccesses increments authentication successes
func (m *Metrics) IncrementAuthenticationSuccesses() {
	m.AuthenticationSuccesses.Add(1)
}

// IncrementAuthenticationFailures increments authentication failures
func (m *Metrics) IncrementAuthenticationFailures() {
	m.AuthenticationFailures.Add(1)
}

// IncrementRegisteredUEs increments registered UEs count
func (m *Metrics) IncrementRegisteredUEs() {
	m.RegisteredUEs.Add(1)
}

// DecrementRegisteredUEs decrements registered UEs count
func (m *Metrics) DecrementRegisteredUEs() {
	m.RegisteredUEs.Add(-1)
}

// IncrementConnectedUEs increments connected UEs count
func (m *Metrics) IncrementConnectedUEs() {
	m.ConnectedUEs.Add(1)
}

// DecrementConnectedUEs decrements connected UEs count
func (m *Metrics) DecrementConnectedUEs() {
	m.ConnectedUEs.Add(-1)
}

// GetStats returns current statistics
func (m *Metrics) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"registration": map[string]interface{}{
			"attempts":  m.RegistrationAttempts.Load(),
			"successes": m.RegistrationSuccesses.Load(),
			"failures":  m.RegistrationFailures.Load(),
		},
		"authentication": map[string]interface{}{
			"attempts":  m.AuthenticationAttempts.Load(),
			"successes": m.AuthenticationSuccesses.Load(),
			"failures":  m.AuthenticationFailures.Load(),
		},
		"ues": map[string]interface{}{
			"registered": m.RegisteredUEs.Load(),
			"connected":  m.ConnectedUEs.Load(),
		},
		"pdu_sessions": map[string]interface{}{
			"establishments": m.PduSessionEstablishments.Load(),
			"releases":       m.PduSessionReleases.Load(),
			"active":         m.ActivePduSessions.Load(),
		},
		"ngap": map[string]interface{}{
			"ng_setup_attempts":       m.NgSetupAttempts.Load(),
			"ng_setup_successes":      m.NgSetupSuccesses.Load(),
			"initial_ue_messages":     m.InitialUeMessages.Load(),
			"uplink_nas_transports":   m.UplinkNasTransports.Load(),
			"downlink_nas_transports": m.DownlinkNasTransports.Load(),
		},
		"errors": map[string]interface{}{
			"nas_decode_errors":  m.NasDecodeErrors.Load(),
			"ngap_decode_errors": m.NgapDecodeErrors.Load(),
		},
	}
}
