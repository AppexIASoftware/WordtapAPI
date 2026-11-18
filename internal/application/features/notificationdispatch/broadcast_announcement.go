package notificationdispatch

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/inbox"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/push"
	"github.com/google/uuid"
)

var (
	ErrEmptyAnnouncementTitle = errors.New("el título del comunicado es requerido")
	ErrEmptyAnnouncementBody  = errors.New("el mensaje del comunicado es requerido")
)

type BroadcastAnnouncementCommand struct {
	AdminID         string                    `json:"-"`
	Title           string                    `json:"title"`
	Body            string                    `json:"body"`
	ActionURL       *string                   `json:"action_url,omitempty"`
	Type            entities.NotificationType `json:"type,omitempty"`
	TargetRole      *entities.UserRole        `json:"target_role,omitempty"`
	TargetCEFRLevel *string                   `json:"target_cefr_level,omitempty"`
}

type BroadcastAnnouncementResult struct {
	RecipientsCount int `json:"recipients_count"`
	DevicesCount    int `json:"devices_count"`
}

type BroadcastAnnouncementHandler struct {
	userRepo           domainRepo.UserRepository
	inboxRepo          inbox.Repository
	deviceRepo         domainRepo.DeviceRepository
	pushSender         push.PushSender
	classroomNotifRepo domainRepo.ClassroomNotificationRepository
}

func NewBroadcastAnnouncementHandler(
	userRepo domainRepo.UserRepository,
	inboxRepo inbox.Repository,
	deviceRepo domainRepo.DeviceRepository,
	pushSender push.PushSender,
) *BroadcastAnnouncementHandler {
	return &BroadcastAnnouncementHandler{
		userRepo:   userRepo,
		inboxRepo:  inboxRepo,
		deviceRepo: deviceRepo,
		pushSender: pushSender,
	}
}

func (h *BroadcastAnnouncementHandler) WithClassroomNotifRepo(repo domainRepo.ClassroomNotificationRepository) *BroadcastAnnouncementHandler {
	h.classroomNotifRepo = repo
	return h
}

func (h *BroadcastAnnouncementHandler) Handle(ctx context.Context, cmd BroadcastAnnouncementCommand) (*BroadcastAnnouncementResult, error) {
	cmd.Title = strings.TrimSpace(cmd.Title)
	cmd.Body = strings.TrimSpace(cmd.Body)
	if cmd.Title == "" {
		return nil, ErrEmptyAnnouncementTitle
	}
	if cmd.Body == "" {
		return nil, ErrEmptyAnnouncementBody
	}

	notifType := cmd.Type
	if notifType == "" {
		notifType = entities.NotificationTypeSystem
	}

	userIDs, err := h.userRepo.ListUserIDs(ctx, cmd.TargetRole, cmd.TargetCEFRLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to list recipient users: %w", err)
	}

	if len(userIDs) == 0 {
		return &BroadcastAnnouncementResult{RecipientsCount: 0, DevicesCount: 0}, nil
	}

	now := time.Now().UTC()
	tokens, err := h.deviceRepo.GetTokensByUsers(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recipient device tokens: %w", err)
	}

	var devicesDispatched atomic.Int64
	if h.pushSender != nil && len(tokens) > 0 {
		data := map[string]string{
			"type": string(notifType),
		}
		if cmd.ActionURL != nil && *cmd.ActionURL != "" {
			data["action_url"] = *cmd.ActionURL
		}

		// Concurrent dispatch using a semaphore bounded to 10 parallel workers
		const maxConcurrency = 10
		sem := make(chan struct{}, maxConcurrency)
		var wg sync.WaitGroup

		for _, token := range tokens {
			wg.Add(1)
			sem <- struct{}{}
			go func(t string) {
				defer wg.Done()
				defer func() { <-sem }()

				sendCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
				defer cancel()

				if sendErr := h.pushSender.Send(sendCtx, t, cmd.Title, cmd.Body, data); sendErr != nil {
					// Log delivery failure to distinguish FCM errors from missing device tokens
					log.Printf("[push] announcement delivery failed: %v", sendErr)
				} else {
					devicesDispatched.Add(1)
				}
			}(token)
		}
		wg.Wait()
	}

	finalStatus := entities.ClassroomNotificationStatusSent
	if len(userIDs) == 0 {
		finalStatus = entities.ClassroomNotificationStatusFailed
	} else if h.pushSender != nil && len(tokens) > 0 && devicesDispatched.Load() == 0 {
		finalStatus = entities.ClassroomNotificationStatusPartial
	}

	// Persist master record BEFORE child inbox entries so classroom_notification_id FK is valid
	var classroomNotifID *string
	adminID := strings.TrimSpace(cmd.AdminID)
	if h.classroomNotifRepo != nil && adminID != "" {
		actionType := "announcement"
		var targetCourseID *string
		var targetLessonID *string
		if cmd.ActionURL != nil && *cmd.ActionURL != "" {
			rawURL := *cmd.ActionURL
			if strings.HasPrefix(rawURL, "/courses/") {
				if cID := cleanEntityIDSegment(strings.TrimPrefix(rawURL, "/courses/")); cID != nil {
					targetCourseID = cID
					actionType = "open_content"
				}
			} else if strings.HasPrefix(rawURL, "/lessons/") {
				if lID := cleanEntityIDSegment(strings.TrimPrefix(rawURL, "/lessons/")); lID != nil {
					targetLessonID = lID
					actionType = "open_content"
				}
			}
		}

		historyID := uuid.NewString()
		history := &entities.ClassroomNotification{
			ID:             historyID,
			ClassroomID:    nil,
			TeacherID:      adminID,
			Title:          cmd.Title,
			Message:        cmd.Body,
			ActionType:     actionType,
			TargetCourseID: targetCourseID,
			TargetLessonID: targetLessonID,
			SentAt:         now,
			RecipientCount: len(userIDs),
			ReadCount:      0,
			Status:         finalStatus,
			CreatedAt:      now,
		}
		if err := h.classroomNotifRepo.Create(ctx, history); err != nil {
			if history.TargetCourseID != nil || history.TargetLessonID != nil {
				history.TargetCourseID = nil
				history.TargetLessonID = nil
				err = h.classroomNotifRepo.Create(ctx, history)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to create notification history: %w", err)
			}
		}
		classroomNotifID = &historyID
	}

	notifications := make([]entities.Notification, len(userIDs))
	for i, uID := range userIDs {
		notifications[i] = entities.Notification{
			ID:                      uuid.NewString(),
			UserID:                  uID,
			ClassroomNotificationID: classroomNotifID,
			NotificationType:        notifType,
			Title:                   cmd.Title,
			Body:                    cmd.Body,
			ActionURL:               cmd.ActionURL,
			SentAt:                  &now,
			CreatedAt:               now,
		}
	}

	if err := h.inboxRepo.CreateBatch(ctx, notifications); err != nil {
		return nil, fmt.Errorf("failed to save announcements in inbox: %w", err)
	}

	return &BroadcastAnnouncementResult{
		RecipientsCount: len(userIDs),
		DevicesCount:    int(devicesDispatched.Load()),
	}, nil
}

func cleanEntityIDSegment(raw string) *string {
	s := strings.TrimSpace(raw)
	if idx := strings.IndexAny(s, "?#"); idx >= 0 {
		s = s[:idx]
	}
	s = strings.Trim(s, "/")
	if s == "" || strings.Contains(s, "/") || len(s) > 36 {
		return nil
	}
	return &s
}
