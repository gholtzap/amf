# AMF

## IMPLEMENTED FEATURES

### NGAP (N2 Interface) Protocol
- NG Setup Request/Response handling
- Initial UE Message handling
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
- GUTI-based registration
- SUCI/SUPI handling

#### Security
- 5G-AKA authentication (via AUSF)
- Re-authentication
- NAS Security Mode Command/Complete
- NIA2 integrity protection
- NEA2 encryption
- Key derivation (Kseaf → Kamf → KnasEnc/KnasInt)
- MAC calculation and verification
- Security context management

#### Other MM Procedures
- Service Request/Accept
- Extended service request
- Deregistration (UE-initiated and Network-initiated)
- Deregistration with re-registration required
- Configuration Update Command/Complete
- GUTI reallocation
- Authentication Request/Response/Reject/Failure
- Identity Request/Response

### NAS - SM (Session Management)
- PDU Session Establishment Request/Accept/Reject
- PDU Session Release Request/Command/Complete
- UL/DL NAS Transport for N1 SM messages
- PDU session context management
- QoS handling (basic)
- Session AMBR allocation
- 5GSM Status
- Always-on PDU session handling

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

#### Namf_EventExposure
- Event subscription Create/Delete/Modify

#### Namf_Location
- Provide Location Info
- Provide Positioning Info
- Cancel Positioning Info

#### Namf_MT
- Provide Domain Selection Info
- Enable UE Reachability
- Enable Group Reachability

#### Namf_MBSCommunication
- MBS N2 Message Transfer

#### Namf_MBSBroadcast
- MBS Broadcast Context Create/Update/Delete

### NF Consumers (Client Interfaces)
- NRF: NF Registration, Heartbeat, Deregistration
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
- Configuration management (JSON-based)
- Logging infrastructure
- Health check endpoint
- Graceful shutdown with NRF deregistration
- PLMN and S-NSSAI support
- TAI tracking

## NOT IMPLEMENTED FEATURES

### NAS MM Procedures

#### Registration
- Emergency registration
- SNPN registration
- Disaster roaming

#### Authentication
- EAP-AKA' authentication

#### Generic UE Configuration Update
- NITZ (Network Identity and Time Zone)
- Service area list updates

#### Mobility
- Intra-AMF mobility (N2 handover)
- Inter-AMF mobility
- Inter-system mobility (4G/5G)
- Idle mode mobility
- Tracking Area Update

#### Connection Management
- Service Request with emergency fallback
- NAS transport reject

#### 5G MM Timers
- T3502, T3510, T3511, T3512, T3513, T3516, T3517, T3519, T3520, T3521, T3522, T3525, T3540, T3550, T3555, T3560, T3565, T3570

### NAS SM Procedures
- PDU Session Authentication
- SSC mode selection
- Network-requested PDU session establishment
- QoS Flow management (detailed)
- Reflective QoS
- MPTCP support

### SBI Services - Missing/Incomplete

#### Namf_Communication
- UE Context transfer result notification
- Non-UE N2 Info Unsubscribe
- AMF Status Change Subscribe/Unsubscribe/Notify
- N2 Info Transfer result notification

#### Namf_EventExposure
- Event notifications (callback implementation)
- Location reporting
- UE reachability events
- Loss of connectivity events
- UE mobility events
- Communication failure events

#### Namf_Location
- Event-driven location reporting
- Periodic location reporting

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
- Reflective QoS control
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
- Power saving mode (PSM)
- eDRX (Extended Discontinuous Reception)
- MICO mode (Mobile Initiated Connection Only)
- Restricted service area
- Forbidden area management

### Security Features (Not Implemented)
- NIA1/NIA3 integrity algorithms
- NEA0/NEA1/NEA3 ciphering algorithms
- Algorithm negotiation
- Security context synchronization
- Null security
- UE security capabilities matching
- Horizontal key derivation
- Inter-system key derivation

### Database/Persistence
- Subscription data persistence (partial)
- Event subscription restore
- N1N2 subscription persistence
- AMF status subscription persistence
- Backup/restore procedures
- Multi-instance synchronization
