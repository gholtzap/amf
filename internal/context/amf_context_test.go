package context

import (
	"testing"
)

func TestGetAMFContext(t *testing.T) {
	ctx := GetAMFContext()
	if ctx == nil {
		t.Fatal("AMF context should not be nil")
	}

	ctx2 := GetAMFContext()
	if ctx != ctx2 {
		t.Error("AMF context should be singleton")
	}
}

func TestNewUEContext(t *testing.T) {
	ctx := GetAMFContext()
	ranUeNgapId := int64(123)

	ue := ctx.NewUEContext(ranUeNgapId)
	if ue == nil {
		t.Fatal("UE context should not be nil")
	}

	if ue.RanUeNgapId != ranUeNgapId {
		t.Errorf("Expected RAN UE NGAP ID %d, got %d", ranUeNgapId, ue.RanUeNgapId)
	}

	if ue.AmfUeNgapId == 0 {
		t.Error("AMF UE NGAP ID should be allocated")
	}
}

func TestGetUEContext(t *testing.T) {
	ctx := GetAMFContext()
	ranUeNgapId := int64(456)

	ue := ctx.NewUEContext(ranUeNgapId)

	retrievedUE, ok := ctx.GetUEContextByAmfUeNgapId(ue.AmfUeNgapId)
	if !ok {
		t.Fatal("UE context should be found")
	}

	if retrievedUE != ue {
		t.Error("Retrieved UE context should match created UE context")
	}
}

func TestDeleteUEContext(t *testing.T) {
	ctx := GetAMFContext()
	ranUeNgapId := int64(789)

	ue := ctx.NewUEContext(ranUeNgapId)
	amfUeNgapId := ue.AmfUeNgapId

	ctx.DeleteUEContext(amfUeNgapId)

	_, ok := ctx.GetUEContextByAmfUeNgapId(amfUeNgapId)
	if ok {
		t.Error("UE context should be deleted")
	}
}
