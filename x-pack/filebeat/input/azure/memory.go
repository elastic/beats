package azure

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Azure/azure-event-hubs-go"
	"github.com/Azure/azure-event-hubs-go/eph"

	"github.com/Azure/azure-amqp-common-go/uuid"
	"github.com/devigned/tab"

	"github.com/Azure/azure-event-hubs-go/persist"
)

type (
	MemoryLeaserCheckpointer struct {
		store         *SharedStore
		processor     *eph.EventProcessorHost
		leaseDuration time.Duration
		memMu         sync.Mutex
		leases        map[string]*MemoryLease
	}

	MemoryLease struct {
		eph.Lease
		expirationTime time.Time
		Token          string
		Checkpoint     *persist.Checkpoint
		leaser         *MemoryLeaserCheckpointer
	}

	SharedStore struct {
		leases  map[string]*StoreLease
		storeMu sync.Mutex
	}

	StoreLease struct {
		token      string
		expiration time.Time
		ml         *MemoryLease
	}
)

func newMemoryLease(partitionID string) *MemoryLease {
	lease := new(MemoryLease)
	lease.PartitionID = partitionID
	return lease
}

func (s *SharedStore) exists() bool {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	return s.leases != nil
}

func (s *SharedStore) ensure() bool {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if s.leases == nil {
		s.leases = make(map[string]*StoreLease)
	}
	return true
}

func (s *SharedStore) getLease(partitionID string) MemoryLease {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	return *s.leases[partitionID].ml
}

func (s *SharedStore) deleteLease(partitionID string) {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	delete(s.leases, partitionID)
}

func (s *SharedStore) createOrGetLease(partitionID string) MemoryLease {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if _, ok := s.leases[partitionID]; !ok {
		s.leases[partitionID] = new(StoreLease)
	}

	l := s.leases[partitionID]
	if l.ml != nil {
		return *l.ml
	}
	l.ml = newMemoryLease(partitionID)
	return *l.ml
}

func (s *SharedStore) changeLease(partitionID, newToken, oldToken string, duration time.Duration) bool {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if l, ok := s.leases[partitionID]; ok && l.token == oldToken {
		l.token = newToken
		l.expiration = time.Now().Add(duration)
		return true
	}
	return false
}

func (s *SharedStore) releaseLease(partitionID, token string) bool {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if l, ok := s.leases[partitionID]; ok && l.token == token {
		l.token = ""
		l.expiration = time.Now().Add(-1 * time.Second)
		return true
	}
	return false
}

func (s *SharedStore) renewLease(partitionID, token string, duration time.Duration) bool {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if l, ok := s.leases[partitionID]; ok && l.token == token {
		l.expiration = time.Now().Add(duration)
		return true
	}
	return false
}

func (s *SharedStore) acquireLease(partitionID, newToken string, duration time.Duration) bool {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if l, ok := s.leases[partitionID]; ok && (time.Now().After(l.expiration) || l.token == "") {
		l.token = newToken
		l.expiration = time.Now().Add(duration)
		return true
	}
	return false
}

func (s *SharedStore) storeLease(partitionID, token string, ml MemoryLease) bool {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if l, ok := s.leases[partitionID]; ok && l.token == token {
		l.ml = &ml
		return true
	}
	return false
}

func (s *SharedStore) isLeased(partitionID string) bool {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if l, ok := s.leases[partitionID]; ok {
		if time.Now().After(l.expiration) || l.token == "" {
			return false
		}
		return true
	}
	return false
}

// IsNotOwnedOrExpired indicates that the lease has expired and does not owned by a processor
func (l *MemoryLease) isNotOwnedOrExpired(ctx context.Context) bool {
	return l.IsExpired(ctx) || l.Owner == ""
}

// IsExpired indicates that the lease has expired and is no longer valid
func (l *MemoryLease) IsExpired(_ context.Context) bool {
	return !l.leaser.store.isLeased(l.PartitionID)
}

func (l *MemoryLease) expireAfter(d time.Duration) {
	l.expirationTime = time.Now().Add(d)
}

func NewMemoryLeaserCheckpointer(leaseDuration time.Duration, store *SharedStore) *MemoryLeaserCheckpointer {
	return &MemoryLeaserCheckpointer{
		leaseDuration: leaseDuration,
		leases:        make(map[string]*MemoryLease),
		store:         store,
	}
}

func (ml *MemoryLeaserCheckpointer) SetEventHostProcessor(eph *eph.EventProcessorHost) {
	ml.processor = eph
}

func (ml *MemoryLeaserCheckpointer) StoreExists(ctx context.Context) (bool, error) {
	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.StoreExists")
	defer span.End()

	return ml.store.exists(), nil
}

func (ml *MemoryLeaserCheckpointer) EnsureStore(ctx context.Context) error {
	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.EnsureStore")
	defer span.End()

	ml.store.ensure()
	return nil
}

func (ml *MemoryLeaserCheckpointer) DeleteStore(ctx context.Context) error {
	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.DeleteStore")
	defer span.End()

	return ml.EnsureStore(ctx)
}

func (ml *MemoryLeaserCheckpointer) GetLeases(ctx context.Context) ([]eph.LeaseMarker, error) {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.GetLeases")
	defer span.End()

	partitionIDs := ml.processor.GetPartitionIDs()
	leases := make([]eph.LeaseMarker, len(partitionIDs))
	for idx, partitionID := range partitionIDs {
		lease := ml.store.getLease(partitionID)
		lease.leaser = ml
		leases[idx] = &lease
	}
	return leases, nil
}

func (ml *MemoryLeaserCheckpointer) EnsureLease(ctx context.Context, partitionID string) (eph.LeaseMarker, error) {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.EnsureLease")
	defer span.End()

	l := ml.store.createOrGetLease(partitionID)
	l.leaser = ml
	return &l, nil
}

func (ml *MemoryLeaserCheckpointer) DeleteLease(ctx context.Context, partitionID string) error {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.DeleteLease")
	defer span.End()

	ml.store.deleteLease(partitionID)
	return nil
}

func (ml *MemoryLeaserCheckpointer) AcquireLease(ctx context.Context, partitionID string) (eph.LeaseMarker, bool, error) {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, ctx := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.AcquireLease")
	defer span.End()

	lease := ml.store.getLease(partitionID)
	lease.leaser = ml
	uuidToken, err := uuid.NewV4()
	if err != nil {
		tab.For(ctx).Error(err)
		return nil, false, err
	}

	newToken := uuidToken.String()
	if ml.store.isLeased(partitionID) {
		// is leased by someone else due to a race to acquire
		if !ml.store.changeLease(partitionID, newToken, lease.Token, ml.leaseDuration) {
			return nil, false, errors.New("failed to change lease")
		}
	} else {
		if !ml.store.acquireLease(partitionID, newToken, ml.leaseDuration) {
			return nil, false, errors.New("failed to acquire lease")
		}
	}

	lease.Token = newToken
	lease.Owner = ml.processor.GetName()
	lease.IncrementEpoch()
	if !ml.store.storeLease(partitionID, newToken, lease) {
		return nil, false, errors.New("failed to store lease after acquiring or changing")
	}
	ml.leases[partitionID] = &lease
	return &lease, true, nil
}

func (ml *MemoryLeaserCheckpointer) RenewLease(ctx context.Context, partitionID string) (eph.LeaseMarker, bool, error) {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.RenewLease")
	defer span.End()

	lease, ok := ml.leases[partitionID]
	if !ok {
		return nil, false, errors.New("lease was not found")
	}

	if !ml.store.renewLease(partitionID, lease.Token, ml.leaseDuration) {
		return nil, false, errors.New("unable to renew lease")
	}
	return lease, true, nil
}

func (ml *MemoryLeaserCheckpointer) ReleaseLease(ctx context.Context, partitionID string) (bool, error) {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.ReleaseLease")
	defer span.End()

	lease, ok := ml.leases[partitionID]
	if !ok {
		return false, errors.New("lease was not found")
	}

	if !ml.store.releaseLease(partitionID, lease.Token) {
		return false, errors.New("could not release the lease")
	}
	delete(ml.leases, partitionID)
	return true, nil
}

func (ml *MemoryLeaserCheckpointer) UpdateLease(ctx context.Context, partitionID string) (eph.LeaseMarker, bool, error) {
	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.UpdateLease")
	defer span.End()

	lease, ok := ml.leases[partitionID]
	if !ok {
		return nil, false, errors.New("lease was not found")
	}

	if !ml.store.renewLease(partitionID, lease.Token, ml.leaseDuration) {
		return nil, false, errors.New("unable to renew lease")
	}

	if !ml.store.storeLease(partitionID, lease.Token, *lease) {
		return nil, false, errors.New("unable to store lease after renewal")
	}

	return lease, true, nil
}

func (ml *MemoryLeaserCheckpointer) GetCheckpoint(ctx context.Context, partitionID string) (persist.Checkpoint, bool) {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryCheckpointer.GetCheckpoint")
	defer span.End()

	lease, ok := ml.leases[partitionID]
	if ok {
		return *lease.Checkpoint, ok
	}
	return persist.NewCheckpointFromStartOfStream(), ok
}

func (ml *MemoryLeaserCheckpointer) EnsureCheckpoint(ctx context.Context, partitionID string) (persist.Checkpoint, error) {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryCheckpointer.EnsureCheckpoint")
	defer span.End()

	lease, ok := ml.leases[partitionID]
	if ok {
		if lease.Checkpoint == nil {
			checkpoint := persist.NewCheckpointFromStartOfStream()
			lease.Checkpoint = &checkpoint
		}
		return *lease.Checkpoint, nil
	}
	return persist.NewCheckpointFromStartOfStream(), nil
}

func (ml *MemoryLeaserCheckpointer) UpdateCheckpoint(ctx context.Context, partitionID string, checkpoint persist.Checkpoint) error {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryCheckpointer.UpdateCheckpoint")
	defer span.End()

	lease, ok := ml.leases[partitionID]
	if !ok {
		return errors.New("lease for partition isn't owned by this EventProcessorHost")
	}

	lease.Checkpoint = &checkpoint
	if !ml.store.storeLease(partitionID, lease.Token, *lease) {
		return errors.New("could not store lease on update of checkpoint")
	}
	return nil
}

func (ml *MemoryLeaserCheckpointer) DeleteCheckpoint(ctx context.Context, partitionID string) error {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryCheckpointer.DeleteCheckpoint")
	defer span.End()

	lease, ok := ml.leases[partitionID]
	if !ok {
		return errors.New("lease for partition isn't owned by this EventProcessorHost")
	}

	checkpoint := persist.NewCheckpointFromStartOfStream()
	lease.Checkpoint = &checkpoint
	if !ml.store.storeLease(partitionID, lease.Token, *lease) {
		return errors.New("failed to store deleted checkpoint")
	}
	ml.leases[partitionID] = lease
	return nil
}

func (ml *MemoryLeaserCheckpointer) Close() error {
	return nil
}

func startConsumerSpanFromContext(ctx context.Context, operationName string) (tab.Spanner, context.Context) {
	ctx, span := tab.StartSpan(ctx, operationName)
	eventhub.ApplyComponentInfo(span)
	span.AddAttributes(tab.StringAttribute("span.kind", "consumer"))
	return span, ctx
}
