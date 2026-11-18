package commands

import (
	"context"
	"errors"
	"fmt"
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
	ErrEmptyClassroomID      = errors.New("el id del aula es requerido")
	ErrEmptyClassroomTitle   = errors.New("el título del aviso es requerido")
	ErrEmptyClassroomMessage = errors.New("el mensaje del aviso es requerido")
	ErrClassroomNotFound     = errors.New("aula no encontrada")
	ErrForbiddenTeacher      = errors.New("no tienes permisos para publicar avisos en esta aula")
)

type BroadcastClassroomCommand struct {
	ClassroomID   string `json:"classroom_id"`
	TeacherID     string `json:"teacher_id"`
	Title         string `json:"title"`
	Message       string `json:"message"`
	ActionType    string `json:"action_type,omitempty"`
	ActionPayload string `json:"action_payload,omitempty"`
}

type BroadcastClassroomResult struct {
	ClassroomID     string `json:"classroom_id"`
	RecipientsCount int    `json:"recipients_count"`
	DevicesCount    int    `json:"devices_count"`
}

type BroadcastClassroomHandler struct {
	classroomRepo      domainRepo.ClassroomRepository
	inboxRepo          inbox.Repository
	deviceRepo         domainRepo.DeviceRepository
	pushSender         push.PushSender
	classroomNotifRepo domainRepo.ClassroomNotificationRepository
}

func NewBroadcastClassroomHandler(
	classroomRepo domainRepo.ClassroomRepository,
	inboxRepo inbox.Repository,
	deviceRepo domainRepo.DeviceRepository,
	pushSender push.PushSender,
) *BroadcastClassroomHandler {
	return &BroadcastClassroomHandler{
		classroomRepo: classroomRepo,
		inboxRepo:     inboxRepo,
		deviceRepo:    deviceRepo,
		pushSender:    pushSender,
	}
}

// WithClassroomNotifRepo enables saving master notification history for classroom records.
func (h *BroadcastClassroomHandler) WithClassroomNotifRepo(repo domainRepo.ClassroomNotificationRepository) *BroadcastClassroomHandler {
	h.classroomNotifRepo = repo
	return h
}

func (h *BroadcastClassroomHandler) Handle(ctx context.Context, cmd BroadcastClassroomCommand) (*BroadcastClassroomResult, error) {
	cmd.ClassroomID = strings.TrimSpace(cmd.ClassroomID)
	cmd.Title = strings.TrimSpace(cmd.Title)
	cmd.Message = strings.TrimSpace(cmd.Message)

	if cmd.ClassroomID == "" {
		return nil, ErrEmptyClassroomID
	}
	if cmd.Title == "" {
		return nil, ErrEmptyClassroomTitle
	}
	if cmd.Message == "" {
		return nil, ErrEmptyClassroomMessage
	}

	classroom, err := h.classroomRepo.FindByID(ctx, cmd.ClassroomID)
	if err != nil || classroom == nil {
		return nil, ErrClassroomNotFound
	}

	if classroom.TeacherID != cmd.TeacherID {
		return nil, ErrForbiddenTeacher
	}

	enrollments, err := h.classroomRepo.ListEnrollments(ctx, cmd.ClassroomID)
	if err != nil {
		return nil, fmt.Errorf("failed to list classroom enrollments: %w", err)
	}

	if len(enrollments) == 0 {
		return &BroadcastClassroomResult{
			ClassroomID:     cmd.ClassroomID,
			RecipientsCount: 0,
			DevicesCount:    0,
		}, nil
	}

	actionURL := fmt.Sprintf("/classrooms/%s", cmd.ClassroomID)
	if cmd.ActionType == "open_lesson" && strings.TrimSpace(cmd.ActionPayload) != "" {
		actionURL = fmt.Sprintf("/lessons/%s", strings.TrimSpace(cmd.ActionPayload))
	} else if cmd.ActionType == "open_course" && strings.TrimSpace(cmd.ActionPayload) != "" {
		actionURL = fmt.Sprintf("/courses/%s", strings.TrimSpace(cmd.ActionPayload))
	}

	now := time.Now().UTC()
	studentIDs := make([]string, 0, len(enrollments))
	for _, en := range enrollments {
		if en.StudentID == "" {
			continue
		}
		studentIDs = append(studentIDs, en.StudentID)
	}

	tokens, err := h.deviceRepo.GetTokensByUsers(ctx, studentIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch classroom student device tokens: %w", err)
	}

	var devicesDispatched atomic.Int64
	if h.pushSender != nil && len(tokens) > 0 {
		data := map[string]string{
			"type":           "classroom_broadcast",
			"classroom_id":   cmd.ClassroomID,
			"action_type":    cmd.ActionType,
			"action_payload": cmd.ActionPayload,
			"action_url":     actionURL,
		}

		// Parallel push dispatch with bounded worker semaphore pool
		const maxWorkers = 15
		sem := make(chan struct{}, maxWorkers)
		var wg sync.WaitGroup

		for _, token := range tokens {
			wg.Add(1)
			sem <- struct{}{}
			go func(t string) {
				defer wg.Done()
				defer func() { <-sem }()

				sendCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
				defer cancel()

				if sendErr := h.pushSender.Send(sendCtx, t, cmd.Title, cmd.Message, data); sendErr == nil {
					devicesDispatched.Add(1)
				}
			}(token)
		}
		wg.Wait()
	}

	// Persist master record BEFORE child inbox notifications so FK integrity holds on batch insert
	var classroomNotifID *string
	if h.classroomNotifRepo != nil {
		historyStatus := entities.ClassroomNotificationStatusSent
		if len(studentIDs) == 0 {
			historyStatus = entities.ClassroomNotificationStatusFailed
		} else if h.pushSender != nil && len(tokens) > 0 && devicesDispatched.Load() == 0 {
			historyStatus = entities.ClassroomNotificationStatusPartial
		}

		var targetCourseID *string
		var targetLessonID *string
		actionType := "open_classroom"
		if cmd.ActionType == "open_lesson" {
			if lID := cleanEntityIDSegment(cmd.ActionPayload); lID != nil {
				targetLessonID = lID
				actionType = "open_content"
			}
		} else if cmd.ActionType == "open_course" {
			if cID := cleanEntityIDSegment(cmd.ActionPayload); cID != nil {
				targetCourseID = cID
				actionType = "open_content"
			}
		}

		historyID := uuid.NewString()
		history := &entities.ClassroomNotification{
			ID:             historyID,
			ClassroomID:    &cmd.ClassroomID,
			TeacherID:      cmd.TeacherID,
			Title:          cmd.Title,
			Message:        cmd.Message,
			ActionType:     actionType,
			TargetCourseID: targetCourseID,
			TargetLessonID: targetLessonID,
			SentAt:         now,
			RecipientCount: len(studentIDs),
			ReadCount:      0,
			Status:         historyStatus,
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

	notifications := make([]entities.Notification, 0, len(studentIDs))
	for _, sID := range studentIDs {
		notifications = append(notifications, entities.Notification{
			ID:                      uuid.NewString(),
			UserID:                  sID,
			ClassroomNotificationID: classroomNotifID,
			NotificationType:        entities.NotificationTypeReminder,
			Title:                   cmd.Title,
			Body:                    cmd.Message,
			ActionURL:               &actionURL,
			SentAt:                  &now,
			CreatedAt:               now,
		})
	}

	if len(notifications) > 0 {
		if err := h.inboxRepo.CreateBatch(ctx, notifications); err != nil {
			return nil, fmt.Errorf("failed to save classroom notifications in inbox: %w", err)
		}
	}

	return &BroadcastClassroomResult{
		ClassroomID:     cmd.ClassroomID,
		RecipientsCount: len(studentIDs),
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
