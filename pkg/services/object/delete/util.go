package deletesvc

import (
	"errors"

	"github.com/nspcc-dev/neofs-node/pkg/core/object"
	getsvc "github.com/nspcc-dev/neofs-node/pkg/services/object/get"
	putsvc "github.com/nspcc-dev/neofs-node/pkg/services/object/put"
	searchsvc "github.com/nspcc-dev/neofs-node/pkg/services/object/search"
	"github.com/nspcc-dev/neofs-node/pkg/services/object_manager/placement"
	objectSDK "github.com/nspcc-dev/neofs-sdk-go/object"
	addressSDK "github.com/nspcc-dev/neofs-sdk-go/object/address"
	oidSDK "github.com/nspcc-dev/neofs-sdk-go/object/id"
)

type headSvcWrapper getsvc.Service

type searchSvcWrapper searchsvc.Service

type putSvcWrapper putsvc.Service

type simpleIDWriter struct {
	ids []*oidSDK.ID
}

func (w *headSvcWrapper) headAddress(exec *execCtx, addr *addressSDK.Address) (*object.Object, error) {
	wr := getsvc.NewSimpleObjectWriter()

	p := getsvc.HeadPrm{}
	p.SetCommonParameters(exec.commonParameters())
	p.SetHeaderWriter(wr)
	p.WithRawFlag(true)
	p.WithAddress(addr)

	err := (*getsvc.Service)(w).Head(exec.context(), p)
	if err != nil {
		return nil, err
	}

	return wr.Object(), nil
}

func (w *headSvcWrapper) splitInfo(exec *execCtx) (*objectSDK.SplitInfo, error) {
	_, err := w.headAddress(exec, exec.address())

	var errSplitInfo *objectSDK.SplitInfoError

	switch {
	case err == nil:
		return nil, nil
	case errors.As(err, &errSplitInfo):
		return errSplitInfo.SplitInfo(), nil
	default:
		return nil, err
	}
}

func (w *headSvcWrapper) children(exec *execCtx) ([]*oidSDK.ID, error) {
	a := exec.newAddress(exec.splitInfo.Link())

	linking, err := w.headAddress(exec, a)
	if err != nil {
		return nil, err
	}

	return linking.Children(), nil
}

func (w *headSvcWrapper) previous(exec *execCtx, id *oidSDK.ID) (*oidSDK.ID, error) {
	a := exec.newAddress(id)

	h, err := w.headAddress(exec, a)
	if err != nil {
		return nil, err
	}

	return h.PreviousID(), nil
}

func (w *searchSvcWrapper) splitMembers(exec *execCtx) ([]*oidSDK.ID, error) {
	fs := objectSDK.SearchFilters{}
	fs.AddSplitIDFilter(objectSDK.MatchStringEqual, exec.splitInfo.SplitID())

	wr := new(simpleIDWriter)

	p := searchsvc.Prm{}
	p.SetWriter(wr)
	p.SetCommonParameters(exec.commonParameters())
	p.WithContainerID(exec.containerID())
	p.WithSearchFilters(fs)

	err := (*searchsvc.Service)(w).Search(exec.context(), p)
	if err != nil {
		return nil, err
	}

	return wr.ids, nil
}

func (s *simpleIDWriter) WriteIDs(ids []*oidSDK.ID) error {
	s.ids = append(s.ids, ids...)

	return nil
}

func (w *putSvcWrapper) put(exec *execCtx, broadcast bool) (*oidSDK.ID, error) {
	streamer, err := (*putsvc.Service)(w).Put(exec.context())
	if err != nil {
		return nil, err
	}

	payload := exec.tombstoneObj.Payload()

	initPrm := new(putsvc.PutInitPrm).
		WithCommonPrm(exec.commonParameters()).
		WithObject(exec.tombstoneObj.CutPayload())

	if broadcast {
		initPrm.WithTraverseOption(placement.WithoutSuccessTracking())
	}

	err = streamer.Init(initPrm)
	if err != nil {
		return nil, err
	}

	err = streamer.SendChunk(new(putsvc.PutChunkPrm).WithChunk(payload))
	if err != nil {
		return nil, err
	}

	r, err := streamer.Close()
	if err != nil {
		return nil, err
	}

	return r.ObjectID(), nil
}
