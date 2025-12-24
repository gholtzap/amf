# AMF

## IMPLEMENTED FEATURES

### NGAP (N2 Interface) Protocol
- NG Setup Request/Response handling with TAI validation
- Initial UE Message handling

## NOT IMPLEMENTED FEATURES

### NGAP (N2 Interface) Protocol
- Uplink/Downlink NAS Transport
- Initial Context Setup Response
- PDU Session Resource Setup Request/Response
- UE Context Release Request
- Path Switch Request
- RAN context management
- Paging
- NG Reset
- Error Indication handling
- Overload Start/Stop
- AMF Configuration Update
- RAN Configuration Update
- UE Radio Capability Management
- UE TNLA Binding Release
- Trace Start
- Deactivate Trace
- Write-Replace Warning
- Location Reporting
- Cell Traffic Trace
- Handover Required
- Handover Request/Acknowledge
- Handover Command
- Handover Notify
- RAN CP Relocation Indication

### NAS (N1 Interface) Protocol - MM (Mobility Management)

#### Registration Management
- Initial Registration
- Mobility Registration Update
- Periodic Registration Update
- Emergency registration
- GUTI-based registration
- SUCI/SUPI handling

#### Security
- 5G-AKA authentication (via AUSF)
- Re-authentication
- NAS Security Mode Command/Complete
- NIA0/NIA1/NIA2/NIA3 integrity protection
- NEA0/NEA1/NEA2/NEA3 encryption
- Key derivation (Kseaf → Kamf → KnasEnc/KnasInt)
- MAC calculation and verification
- Security context management
- Algorithm negotiation
- UE security capabilities matching

#### Mobility
- Idle mode mobility (RAN Notification Area update)

#### Other MM Procedures
- Service Request/Accept
- Extended service request
- Deregistration (UE-initiated and Network-initiated)
- Deregistration with re-registration required
- Configuration Update Command/Complete
- Generic UE Configuration Update
- GUTI reallocation
- Service area list updates
- NITZ (Network Identity and Time Zone)
- Authentication Request/Response/Reject/Failure
- Identity Request/Response
- Tracking Area Update
- 5GMM Status (NAS transport reject)

#### 5G MM Timers
- T3502 (Registration Reject timer)
- T3510 (Registration Request timer with retransmission)
- T3511 (Registration failure timer)
- T3512 (Periodic Registration Update timer)
- T3513 (Paging timer with retransmission)
- T3516 (5GMM status timer with retransmission)
- T3517 (Service Accept timer with retransmission)
- T3519 (Notification timer with retransmission)
- T3520 (GUTI reallocation timer with retransmission)
- T3521 (Deregistration request timer - UE-initiated)
- T3522 (Deregistration timer with retransmission)
- T3525 (Identity response timer)
- T3540 (Deregistration request timer with retransmission)
- T3550 (Registration Accept timer with retransmission)
- T3555 (Configuration Update Command timer with retransmission)
- T3560 (Authentication Request timer with retransmission)
- T3565 (Security Mode Command timer with retransmission)
- T3570 (Identity Request timer with retransmission)

### NAS - SM (Session Management)
- PDU Session Establishment Request/Accept/Reject
- Network-requested PDU session establishment
- PDU Session Authentication (EAP-AKA')
- PDU Session Release Request/Command/Complete
- UL/DL NAS Transport for N1 SM messages
- PDU session context management
- QoS handling (basic)
- QoS Flow management (detailed)
  - QoS flow creation, modification, and deletion
  - QoS flow description parsing and encoding
  - QoS rule parsing and encoding
  - Network-initiated QoS flow modification
  - UE-initiated QoS flow modification requests
- Session AMBR allocation
- 5GSM Status
- Always-on PDU session handling
- SSC mode selection
- MPTCP support
- Reflective QoS

### SBI (Service-Based Interfaces)

#### Namf_Communication
- UE Context Create/Get/Release
- Query UE Contexts
- N1N2 Message Transfer
- N1N2 Message Subscriptions
- EBI Assignment
- UE Context Transfer
- Registration Status Update
- UE Context Relocation
- Cancel UE Context Relocation
- Non-UE N2 Message Transfer subscriptions
- AMF Status subscriptions
- AMF Status Change Subscribe/Unsubscribe/Notify
- N2 Info Transfer result notification
- UE Context transfer result notification

#### Namf_EventExposure
- Event subscription Create/Delete/Modify
- Event notifications (callback implementation)
  - Registration state report
  - Connectivity state report
  - Reachability report
  - Loss of connectivity events
  - Communication failure events
  - UE mobility events
  - Location reporting (periodic and event-driven)

#### Namf_Location
- Provide Location Info2
- Provide Positioning Info
- Cancel Positioning Info
- Periodic location reporting
- Event-driven location reporting (area-based and motion-based)

#### Namf_MT
- Provide Domain Selection Info
- Enable UE Reachability
- Enable Group Reachability

#### Namf_MBSCommunication
- MBS N2 Message Transfer

#### Namf_MBSBroadcast
- MBS Broadcast Context Create/Update/Delete

### NF Consumers (Client Interfaces)
- NRF: NF Registration, Heartbeat, Deregistration, Auto-recovery on heartbeat failures
- AUSF: Authentication request/confirmation
- UDM: AM subscription data retrieval, AMF registration
- SMF: SM Context Create

### Core Features
- UE context management (NGAP IDs, state tracking)
- RAN context management
- GUTI allocation and management (TMSI counter)
- Multi-UE support
- MongoDB persistence for UE contexts and subscriptions
- Database restore on startup
- Event subscription persistence and restore
- N1N2 subscription persistence and restore
- AMF status subscription persistence and restore
- Non-UE N2 subscription persistence and restore
- Database backup/restore procedures
- Configuration management (JSON-based)
- Logging infrastructure
- Health check endpoint
- Graceful shutdown with NRF deregistration
- PLMN and S-NSSAI support
- TAI tracking
- MICO mode (Mobile Initiated Connection Only)
- eDRX (Extended Discontinuous Reception) power saving
- PSM (Power Saving Mode) with T3324 and T3412 extended timers
- Forbidden area management
- Restricted service area

### NAS MM Procedures

#### Registration
- SNPN registration
- Disaster roaming

#### Authentication
- EAP-AKA' authentication

#### Mobility
- Intra-AMF mobility (N2 handover)
- Inter-AMF mobility
- Inter-system mobility (4G/5G)

#### Connection Management
- Service Request with emergency fallback


### SBI Services - Missing/Incomplete

#### Namf_Communication
- Non-UE N2 Info Unsubscribe

#### Namf_MT
- SMS delivery over NAS

### Advanced Features
- Network Slicing (advanced S-NSSAI management)
- Access traffic steering/switching/splitting (ATSSS)
- Emergency services support (eCall, emergency sessions)
- LADN (Local Area Data Network) support
- Edge computing support (MEC)
- Dual Registration (4G+5G)
- Multi-access PDU connectivity service
- UE Policy Delivery Service
- SMS over NAS
- LPP (Location Positioning Protocol)
- SoR (Steering of Roaming)
- UE Parameter Update
- CAG (Closed Access Group) support
- Network verification support
- SUCI concealment/decryption
- Protection of initial NAS message
- Service gap control
- Extended DRX

### Security Features (Not Implemented)
- Security context synchronization
- Null security
- Horizontal key derivation
- Inter-system key derivation

### Database/Persistence
- Multi-instance synchronization
