package database

import (
	"context"
	"fmt"
	"time"

	amfcontext "github.com/gavin/amf/internal/context"
	"github.com/gavin/amf/internal/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UERepository struct {
	collection *mongo.Collection
	ctx        context.Context
}

type UEContextDocument struct {
	AmfUeNgapId           int64                                  `bson:"amf_ue_ngap_id"`
	RanUeNgapId           int64                                  `bson:"ran_ue_ngap_id"`
	Supi                  string                                 `bson:"supi,omitempty"`
	Suci                  string                                 `bson:"suci,omitempty"`
	Pei                   string                                 `bson:"pei,omitempty"`
	Guti                  *GutiDocument                          `bson:"guti,omitempty"`
	RegistrationType      uint8                                  `bson:"registration_type"`
	NgKsi                 int                                    `bson:"ng_ksi"`
	AuthenticationCtxId   string                                 `bson:"authentication_ctx_id,omitempty"`
	UeSecurityCapability  string                                 `bson:"ue_security_capability,omitempty"`
	ULCount               uint32                                 `bson:"ul_count"`
	DLCount               uint32                                 `bson:"dl_count"`
	RegistrationState     string                                 `bson:"registration_state"`
	Tai                   TaiDocument                            `bson:"tai"`
	CellId                string                                 `bson:"cell_id,omitempty"`
	SecurityContext       *SecurityContextDocument               `bson:"security_context,omitempty"`
	PduSessions           map[int32]*PduSessionContextDocument   `bson:"pdu_sessions,omitempty"`
	AccessType            string                                 `bson:"access_type,omitempty"`
	CmState               string                                 `bson:"cm_state"`
	RmState               string                                 `bson:"rm_state"`
	UpdatedAt             time.Time                              `bson:"updated_at"`
}

type TaiDocument struct {
	PlmnId PlmnIdDocument `bson:"plmn_id"`
	Tac    string         `bson:"tac"`
}

type PlmnIdDocument struct {
	Mcc string `bson:"mcc"`
	Mnc string `bson:"mnc"`
}

type GutiDocument struct {
	PlmnId      PlmnIdDocument `bson:"plmn_id"`
	AmfRegionId string         `bson:"amf_region_id"`
	AmfSetId    string         `bson:"amf_set_id"`
	AmfPointer  string         `bson:"amf_pointer"`
	Tmsi        uint32         `bson:"tmsi"`
}

type SecurityContextDocument struct {
	Kseaf              []byte `bson:"kseaf,omitempty"`
	Kamf               []byte `bson:"kamf,omitempty"`
	KnasInt            []byte `bson:"knas_int,omitempty"`
	KnasEnc            []byte `bson:"knas_enc,omitempty"`
	NgKsi              int    `bson:"ng_ksi"`
	IntegrityAlg       int    `bson:"integrity_alg"`
	CipheringAlg       int    `bson:"ciphering_alg"`
	IntegrityAlgorithm int    `bson:"integrity_algorithm"`
	CipheringAlgorithm int    `bson:"ciphering_algorithm"`
	Activated          bool   `bson:"activated"`
}

type PduSessionContextDocument struct {
	PduSessionId int32       `bson:"pdu_session_id"`
	Dnn          string      `bson:"dnn"`
	Snssai       SnssaiDocument `bson:"snssai"`
	State        string      `bson:"state"`
	SmContextRef string      `bson:"sm_context_ref,omitempty"`
	SmContextId  string      `bson:"sm_context_id,omitempty"`
}

type SnssaiDocument struct {
	Sst int32  `bson:"sst"`
	Sd  string `bson:"sd,omitempty"`
}

func NewUERepository(client *MongoDBClient) *UERepository {
	collection := client.GetCollection("ue_contexts")

	ctx := client.Context()

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "amf_ue_ngap_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		logger.InitLog.Warnf("Failed to create index on ue_contexts: %v", err)
	}

	logger.InitLog.Info("UE Repository initialized")

	return &UERepository{
		collection: collection,
		ctx:        ctx,
	}
}

func (r *UERepository) Save(ue *amfcontext.UEContext) error {
	doc := r.toDocument(ue)

	filter := bson.M{"amf_ue_ngap_id": ue.AmfUeNgapId}
	opts := options.Replace().SetUpsert(true)

	_, err := r.collection.ReplaceOne(r.ctx, filter, doc, opts)
	if err != nil {
		return fmt.Errorf("failed to save UE context: %w", err)
	}

	logger.DbLog.Debugf("UE context saved to MongoDB, AMF UE NGAP ID: %d", ue.AmfUeNgapId)
	return nil
}

func (r *UERepository) FindByAmfUeNgapId(id int64) (*amfcontext.UEContext, error) {
	var doc UEContextDocument

	filter := bson.M{"amf_ue_ngap_id": id}
	err := r.collection.FindOne(r.ctx, filter).Decode(&doc)

	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find UE context: %w", err)
	}

	ue := r.fromDocument(&doc)
	logger.DbLog.Debugf("UE context loaded from MongoDB, AMF UE NGAP ID: %d", id)
	return ue, nil
}

func (r *UERepository) FindAll() ([]*amfcontext.UEContext, error) {
	cursor, err := r.collection.Find(r.ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find UE contexts: %w", err)
	}
	defer cursor.Close(r.ctx)

	var ueContexts []*amfcontext.UEContext
	for cursor.Next(r.ctx) {
		var doc UEContextDocument
		if err := cursor.Decode(&doc); err != nil {
			logger.DbLog.Warnf("Failed to decode UE context: %v", err)
			continue
		}
		ueContexts = append(ueContexts, r.fromDocument(&doc))
	}

	logger.DbLog.Infof("Loaded %d UE contexts from MongoDB", len(ueContexts))
	return ueContexts, nil
}

func (r *UERepository) Delete(amfUeNgapId int64) error {
	filter := bson.M{"amf_ue_ngap_id": amfUeNgapId}

	_, err := r.collection.DeleteOne(r.ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete UE context: %w", err)
	}

	logger.DbLog.Debugf("UE context deleted from MongoDB, AMF UE NGAP ID: %d", amfUeNgapId)
	return nil
}

func (r *UERepository) toDocument(ue *amfcontext.UEContext) *UEContextDocument {
	var gutiDoc *GutiDocument
	if ue.Guti != nil {
		gutiDoc = &GutiDocument{
			PlmnId: PlmnIdDocument{
				Mcc: ue.Guti.PlmnId.Mcc,
				Mnc: ue.Guti.PlmnId.Mnc,
			},
			AmfRegionId: ue.Guti.AmfRegionId,
			AmfSetId:    ue.Guti.AmfSetId,
			AmfPointer:  ue.Guti.AmfPointer,
			Tmsi:        ue.Guti.Tmsi,
		}
	}

	doc := &UEContextDocument{
		AmfUeNgapId:          ue.AmfUeNgapId,
		RanUeNgapId:          ue.RanUeNgapId,
		Supi:                 ue.Supi,
		Suci:                 ue.Suci,
		Pei:                  ue.Pei,
		Guti:                 gutiDoc,
		RegistrationType:     ue.RegistrationType,
		NgKsi:                ue.NgKsi,
		AuthenticationCtxId:  ue.AuthenticationCtxId,
		UeSecurityCapability: ue.UeSecurityCapability,
		ULCount:              ue.ULCount,
		DLCount:              ue.DLCount,
		RegistrationState:    string(ue.RegistrationState),
		Tai: TaiDocument{
			PlmnId: PlmnIdDocument{
				Mcc: ue.Tai.PlmnId.Mcc,
				Mnc: ue.Tai.PlmnId.Mnc,
			},
			Tac: ue.Tai.Tac,
		},
		CellId:     ue.CellId,
		AccessType: string(ue.AccessType),
		CmState:    string(ue.CmState),
		RmState:    string(ue.RmState),
		UpdatedAt:  time.Now(),
	}

	if ue.SecurityContext != nil {
		doc.SecurityContext = &SecurityContextDocument{
			Kseaf:              ue.SecurityContext.Kseaf,
			Kamf:               ue.SecurityContext.Kamf,
			KnasInt:            ue.SecurityContext.KnasInt,
			KnasEnc:            ue.SecurityContext.KnasEnc,
			NgKsi:              ue.SecurityContext.NgKsi,
			IntegrityAlg:       ue.SecurityContext.IntegrityAlg,
			CipheringAlg:       ue.SecurityContext.CipheringAlg,
			IntegrityAlgorithm: ue.SecurityContext.IntegrityAlgorithm,
			CipheringAlgorithm: ue.SecurityContext.CipheringAlgorithm,
			Activated:          ue.SecurityContext.Activated,
		}
	}

	if ue.PduSessions != nil {
		doc.PduSessions = make(map[int32]*PduSessionContextDocument)
		for id, session := range ue.PduSessions {
			doc.PduSessions[id] = &PduSessionContextDocument{
				PduSessionId: session.PduSessionId,
				Dnn:          session.Dnn,
				Snssai: SnssaiDocument{
					Sst: session.Snssai.Sst,
					Sd:  session.Snssai.Sd,
				},
				State:        string(session.State),
				SmContextRef: session.SmContextRef,
				SmContextId:  session.SmContextId,
			}
		}
	}

	return doc
}

func (r *UERepository) fromDocument(doc *UEContextDocument) *amfcontext.UEContext {
	var guti *amfcontext.Guti
	if doc.Guti != nil {
		guti = &amfcontext.Guti{
			PlmnId: amfcontext.PlmnId{
				Mcc: doc.Guti.PlmnId.Mcc,
				Mnc: doc.Guti.PlmnId.Mnc,
			},
			AmfRegionId: doc.Guti.AmfRegionId,
			AmfSetId:    doc.Guti.AmfSetId,
			AmfPointer:  doc.Guti.AmfPointer,
			Tmsi:        doc.Guti.Tmsi,
		}
	}

	ue := &amfcontext.UEContext{
		AmfUeNgapId:          doc.AmfUeNgapId,
		RanUeNgapId:          doc.RanUeNgapId,
		Supi:                 doc.Supi,
		Suci:                 doc.Suci,
		Pei:                  doc.Pei,
		Guti:                 guti,
		RegistrationType:     doc.RegistrationType,
		NgKsi:                doc.NgKsi,
		AuthenticationCtxId:  doc.AuthenticationCtxId,
		UeSecurityCapability: doc.UeSecurityCapability,
		ULCount:              doc.ULCount,
		DLCount:              doc.DLCount,
		RegistrationState:    amfcontext.RegistrationState(doc.RegistrationState),
		Tai: amfcontext.Tai{
			PlmnId: amfcontext.PlmnId{
				Mcc: doc.Tai.PlmnId.Mcc,
				Mnc: doc.Tai.PlmnId.Mnc,
			},
			Tac: doc.Tai.Tac,
		},
		CellId:     doc.CellId,
		AccessType: amfcontext.AccessType(doc.AccessType),
		CmState:    amfcontext.CmState(doc.CmState),
		RmState:    amfcontext.RmState(doc.RmState),
	}

	if doc.SecurityContext != nil {
		ue.SecurityContext = &amfcontext.SecurityContext{
			Kseaf:              doc.SecurityContext.Kseaf,
			Kamf:               doc.SecurityContext.Kamf,
			KnasInt:            doc.SecurityContext.KnasInt,
			KnasEnc:            doc.SecurityContext.KnasEnc,
			NgKsi:              doc.SecurityContext.NgKsi,
			IntegrityAlg:       doc.SecurityContext.IntegrityAlg,
			CipheringAlg:       doc.SecurityContext.CipheringAlg,
			IntegrityAlgorithm: doc.SecurityContext.IntegrityAlgorithm,
			CipheringAlgorithm: doc.SecurityContext.CipheringAlgorithm,
			Activated:          doc.SecurityContext.Activated,
		}
	} else {
		ue.SecurityContext = &amfcontext.SecurityContext{}
	}

	if doc.PduSessions != nil {
		ue.PduSessions = make(map[int32]*amfcontext.PduSessionContext)
		for id, sessionDoc := range doc.PduSessions {
			ue.PduSessions[id] = &amfcontext.PduSessionContext{
				PduSessionId: sessionDoc.PduSessionId,
				Dnn:          sessionDoc.Dnn,
				Snssai: amfcontext.Snssai{
					Sst: sessionDoc.Snssai.Sst,
					Sd:  sessionDoc.Snssai.Sd,
				},
				State:        amfcontext.PduSessionState(sessionDoc.State),
				SmContextRef: sessionDoc.SmContextRef,
				SmContextId:  sessionDoc.SmContextId,
				QosFlows:     make(map[int]*amfcontext.QosFlow),
			}
		}
	} else {
		ue.PduSessions = make(map[int32]*amfcontext.PduSessionContext)
	}

	return ue
}
