package service

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/model"
	"gorm.io/gorm"
)

const (
	maxTicketTitleRunes   = 255
	maxTicketContentRunes = 10000
)

type TicketSummary struct {
	Id        int    `json:"id"`
	Title     string `json:"title"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type TicketMessage struct {
	Id       int    `json:"id"`
	Type     string `json:"type"`
	Role     string `json:"role"`
	Username string `json:"username"`
	Content  string `json:"content,omitempty"`
	Value    string `json:"value,omitempty"`
	Time     int64  `json:"time"`
}

type TicketDetail struct {
	Id        int             `json:"id"`
	Title     string          `json:"title"`
	Type      string          `json:"type"`
	Status    string          `json:"status"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at"`
	ClosedAt  int64           `json:"closed_at"`
	Messages  []TicketMessage `json:"messages"`
}

type CreateTicketInput struct {
	UserId   int
	Username string
	Role     int
	Title    string
	Type     string
	Content  string
}

type ReplyTicketInput struct {
	TicketId  int
	UserId    int
	Username  string
	Role      int
	Content   string
	NewStatus string
}

func ListUserTickets(userId int, page, pageSize int, status string, keyword string) ([]TicketSummary, int64, error) {
	filter, err := buildTicketListFilter(userId, page, pageSize, status, keyword)
	if err != nil {
		return nil, 0, err
	}
	tickets, total, err := model.ListUserTickets(filter)
	if err != nil {
		return nil, 0, err
	}
	return buildTicketSummaries(tickets), total, nil
}

func ListAdminTickets(role int, page, pageSize int, status string, keyword string) ([]TicketSummary, int64, error) {
	if !canManageAllTickets(role) {
		return nil, 0, errors.New("无权进行此操作")
	}
	filter, err := buildTicketListFilter(0, page, pageSize, status, keyword)
	if err != nil {
		return nil, 0, err
	}
	tickets, total, err := model.ListAdminTickets(filter)
	if err != nil {
		return nil, 0, err
	}
	return buildTicketSummaries(tickets), total, nil
}

func CreateTicket(input CreateTicketInput) (*TicketDetail, error) {
	title, err := validateTicketText(input.Title, maxTicketTitleRunes, "工单标题不能为空", "工单标题过长")
	if err != nil {
		return nil, err
	}
	content, err := validateTicketText(input.Content, maxTicketContentRunes, "工单内容不能为空", "工单内容过长")
	if err != nil {
		return nil, err
	}
	ticketType, err := model.ParseTicketType(input.Type)
	if err != nil {
		return nil, err
	}

	ticket := model.NewTicket(title, input.UserId, ticketType)
	entry := &model.TicketEntry{
		EntryType:    model.TicketEntryTypeMessage,
		SenderUserId: input.UserId,
		SenderName:   input.Username,
		SenderRole:   input.Role,
		Content:      content,
		CreatedAt:    ticket.CreatedAt,
	}

	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.CreateTicketTx(tx, ticket); err != nil {
			return err
		}
		entry.TicketId = ticket.Id
		if err := model.CreateTicketEntryTx(tx, entry); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, errors.New("创建工单失败")
	}

	return buildTicketDetail(ticket, []*model.TicketEntry{entry}), nil
}

func GetTicketDetail(ticketId int, userId int, role int) (*TicketDetail, error) {
	ticket, err := model.GetTicketByID(ticketId)
	if err != nil {
		return nil, err
	}
	if err := ensureTicketAccess(ticket, userId, role); err != nil {
		return nil, err
	}
	entries, err := model.GetTicketEntries(ticketId)
	if err != nil {
		return nil, err
	}
	return buildTicketDetail(ticket, entries), nil
}

func ReplyTicket(input ReplyTicketInput) error {
	content, err := validateTicketText(input.Content, maxTicketContentRunes, "回复内容不能为空", "回复内容过长")
	if err != nil {
		return err
	}

	return model.DB.Transaction(func(tx *gorm.DB) error {
		ticket, err := model.GetTicketByIDForUpdate(tx, input.TicketId)
		if err != nil {
			return err
		}
		if err := ensureTicketAccess(ticket, input.UserId, input.Role); err != nil {
			return err
		}

		now := common.GetTimestamp()
		entry := &model.TicketEntry{
			TicketId:     ticket.Id,
			EntryType:    model.TicketEntryTypeMessage,
			SenderUserId: input.UserId,
			SenderName:   input.Username,
			SenderRole:   input.Role,
			Content:      content,
			CreatedAt:    now,
		}
		if err := model.CreateTicketEntryTx(tx, entry); err != nil {
			return errors.New("发送回复失败")
		}
		if err := model.UpdateTicketFieldsTx(tx, ticket.Id, map[string]any{"updated_at": now}); err != nil {
			return errors.New("发送回复失败")
		}
		return nil
	})
}

func CloseTicket(ticketId int, userId int, role int, username string) error {
	return changeTicketStatus(ticketId, userId, role, username, model.TicketStatusCompleted)
}

func UpdateTicketStatus(ticketId int, userId int, role int, username string, status string) error {
	if !canManageAllTickets(role) {
		return errors.New("无权进行此操作")
	}
	targetStatus, err := model.ParseTicketStatus(status)
	if err != nil {
		return err
	}
	return changeTicketStatus(ticketId, userId, role, username, targetStatus)
}

func changeTicketStatus(ticketId int, userId int, role int, username string, targetStatus int) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		ticket, err := model.GetTicketByIDForUpdate(tx, ticketId)
		if err != nil {
			return err
		}
		if err := ensureTicketAccess(ticket, userId, role); err != nil {
			return err
		}
		if ticket.Status == targetStatus {
			return nil
		}

		now := common.GetTimestamp()
		entry := &model.TicketEntry{
			TicketId:     ticket.Id,
			EntryType:    model.TicketEntryTypeStatusChange,
			SenderUserId: userId,
			SenderName:   username,
			SenderRole:   role,
			FromStatus:   ticket.Status,
			ToStatus:     targetStatus,
			CreatedAt:    now,
		}
		if err := model.CreateTicketEntryTx(tx, entry); err != nil {
			return errors.New("更新工单状态失败")
		}

		values := map[string]any{
			"status":     targetStatus,
			"updated_at": now,
			"closed_at":  int64(0),
		}
		if targetStatus == model.TicketStatusCompleted {
			values["closed_at"] = now
		}
		if err := model.UpdateTicketFieldsTx(tx, ticket.Id, values); err != nil {
			return errors.New("更新工单状态失败")
		}
		return nil
	})
}

func buildTicketListFilter(userId int, page int, pageSize int, status string, keyword string) (model.TicketListFilter, error) {
	filter := model.TicketListFilter{
		UserId:  userId,
		Keyword: strings.TrimSpace(keyword),
		Offset:  (page - 1) * pageSize,
		Limit:   pageSize,
	}
	if status == "" || status == "all" {
		return filter, nil
	}
	parsedStatus, err := model.ParseTicketStatus(status)
	if err != nil {
		return model.TicketListFilter{}, err
	}
	filter.Status = parsedStatus
	return filter, nil
}

func buildTicketSummaries(tickets []*model.Ticket) []TicketSummary {
	items := make([]TicketSummary, 0, len(tickets))
	for _, ticket := range tickets {
		items = append(items, TicketSummary{
			Id:        ticket.Id,
			Title:     ticket.Title,
			Type:      model.TicketTypeName(ticket.Type),
			Status:    model.TicketStatusName(ticket.Status),
			CreatedAt: ticket.CreatedAt,
			UpdatedAt: ticket.UpdatedAt,
		})
	}
	return items
}

func buildTicketDetail(ticket *model.Ticket, entries []*model.TicketEntry) *TicketDetail {
	messages := make([]TicketMessage, 0, len(entries))
	for _, entry := range entries {
		message := TicketMessage{
			Id:       entry.Id,
			Username: entry.SenderName,
			Role:     buildMessageRole(entry.SenderRole),
			Time:     entry.CreatedAt,
		}
		if entry.EntryType == model.TicketEntryTypeStatusChange {
			message.Type = "status"
			message.Value = model.TicketStatusName(entry.ToStatus)
		} else {
			message.Type = "message"
			message.Content = entry.Content
		}
		messages = append(messages, message)
	}

	return &TicketDetail{
		Id:        ticket.Id,
		Title:     ticket.Title,
		Type:      model.TicketTypeName(ticket.Type),
		Status:    model.TicketStatusName(ticket.Status),
		CreatedAt: ticket.CreatedAt,
		UpdatedAt: ticket.UpdatedAt,
		ClosedAt:  ticket.ClosedAt,
		Messages:  messages,
	}
}

func buildMessageRole(role int) string {
	if canManageAllTickets(role) {
		return "admin"
	}
	return "user"
}

func ensureTicketAccess(ticket *model.Ticket, userId int, role int) error {
	if canManageAllTickets(role) {
		return nil
	}
	if ticket.UserId != userId {
		return errors.New("无权访问该工单")
	}
	return nil
}

func canManageAllTickets(role int) bool {
	return role >= common.RoleAdminUser
}

func validateTicketText(value string, maxRunes int, emptyMessage string, tooLongMessage string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New(emptyMessage)
	}
	if utf8.RuneCountInString(value) > maxRunes {
		return "", errors.New(tooLongMessage)
	}
	return value, nil
}
