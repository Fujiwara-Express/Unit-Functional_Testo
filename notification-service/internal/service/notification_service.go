package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/notification-service/internal/client"
	"github.com/notification-service/internal/domain"
	"github.com/notification-service/internal/repository"
)

type notificationService struct {
	repo      repository.NotificationRepository
	firebase  client.FirebaseClient
	sendgrid  client.SendGridClient
	whatsapp  client.WhatsAppClient
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(
	repo repository.NotificationRepository,
	firebase client.FirebaseClient,
	sendgrid client.SendGridClient,
	whatsapp client.WhatsAppClient,
) NotificationService {
	return &notificationService{
		repo:     repo,
		firebase: firebase,
		sendgrid: sendgrid,
		whatsapp: whatsapp,
	}
}

func (s *notificationService) SendNotification(ctx context.Context, req SendNotificationInput) (*SendNotificationOutput, error) {
	tmpl, err := s.repo.GetTemplateByID(ctx, req.TemplateID)
	if err != nil {
		return nil, err
	}

	message := domain.RenderTemplate(tmpl.BodyTemplate, req.Variables)

	var providerErr error
	switch req.Channel {
	case domain.ChannelPush:
		providerErr = s.firebase.SendPush(ctx, req.UserID, message)
	case domain.ChannelEmail:
		providerErr = s.sendgrid.SendEmail(ctx, req.UserID, tmpl.Subject, message)
	case domain.ChannelWhatsApp:
		providerErr = s.whatsapp.SendWhatsApp(ctx, req.UserID, message)
	default:
		providerErr = fmt.Errorf("%w: unknown channel %s", domain.ErrValidation, req.Channel)
	}

	status := domain.NotifStatusSent
	if providerErr != nil {
		status = domain.NotifStatusFailed
	}

	notifID := fmt.Sprintf("notif-%d", time.Now().UnixNano())
	log := &domain.NotificationLog{
		NotifID:    notifID,
		UserID:     req.UserID,
		Channel:    req.Channel,
		TemplateID: req.TemplateID,
		Message:    message,
		Status:     status,
		SentAt:     time.Now(),
	}
	createdID, logErr := s.repo.CreateNotificationLog(ctx, log)
	if logErr != nil {
		return nil, logErr
	}

	if providerErr != nil {
		return nil, providerErr
	}

	return &SendNotificationOutput{
		NotificationID: createdID,
		Status:         domain.NotifStatusSent,
		Channel:        req.Channel,
	}, nil
}

func (s *notificationService) ListTemplates(ctx context.Context) ([]*domain.NotificationTemplate, error) {
	return s.repo.ListTemplates(ctx)
}

func (s *notificationService) CreateTemplate(ctx context.Context, req CreateTemplateInput) (*CreateTemplateOutput, error) {
	t := &domain.NotificationTemplate{
		TemplateID:   uuid.New().String(),
		EventType:    req.EventType,
		Channel:      req.Channel,
		Subject:      req.Subject,
		BodyTemplate: req.BodyTemplate,
	}
	id, err := s.repo.CreateTemplate(ctx, t)
	if err != nil {
		return nil, err
	}
	return &CreateTemplateOutput{
		TemplateID: id,
		Status:     "CREATED",
	}, nil
}

func (s *notificationService) UpdateTemplate(ctx context.Context, id string, req UpdateTemplateInput) (*UpdateTemplateOutput, error) {
	if _, err := s.repo.GetTemplateByID(ctx, id); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateTemplate(ctx, id, req.Subject, req.BodyTemplate); err != nil {
		return nil, err
	}
	return &UpdateTemplateOutput{
		TemplateID: id,
		Status:     "UPDATED",
	}, nil
}
