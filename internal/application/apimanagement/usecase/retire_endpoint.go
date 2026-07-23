package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	apimgmtEvent "github.com/gurkanfikretgunak/masterfabric-go/internal/domain/apimanagement/event"
	"github.com/gurkanfikretgunak/masterfabric-go/internal/domain/apimanagement/model"
	"github.com/gurkanfikretgunak/masterfabric-go/internal/domain/apimanagement/repository"
	domainErr "github.com/gurkanfikretgunak/masterfabric-go/internal/shared/errors"
	"github.com/gurkanfikretgunak/masterfabric-go/internal/shared/events"
)

// RetireEndpointUseCase handles retiring an endpoint.
type RetireEndpointUseCase struct {
	endpointRepo repository.EndpointRepository
	eventBus     events.EventBus
}

// NewRetireEndpointUseCase creates a new RetireEndpointUseCase.
func NewRetireEndpointUseCase(endpointRepo repository.EndpointRepository, eventBus events.EventBus) *RetireEndpointUseCase {
	return &RetireEndpointUseCase{endpointRepo: endpointRepo, eventBus: eventBus}
}

// Execute retires an endpoint.
func (uc *RetireEndpointUseCase) Execute(ctx context.Context, endpointID uuid.UUID) error {
	endpoint, err := uc.endpointRepo.GetByID(ctx, endpointID)
	if err != nil {
		return err
	}

	if endpoint.Status == model.EndpointStatusRetired {
		return domainErr.New(domainErr.ErrBadRequest, "endpoint is already retired", nil)
	}

	endpoint.Status = model.EndpointStatusRetired
	if err := uc.endpointRepo.Update(ctx, endpoint); err != nil {
		return err
	}

	// Publish domain event to Kafka
	_ = uc.eventBus.Publish(ctx, events.TopicAPIManagement, apimgmtEvent.EndpointRetired{
		EndpointID: endpointID,
		AppID:      endpoint.AppID,
		Timestamp:  time.Now().UTC(),
	})

	return nil
}
