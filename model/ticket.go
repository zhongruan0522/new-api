package model

import (
	"errors"
	"strconv"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"gorm.io/gorm"
)

const (
	TicketTypeBug = 1 + iota
	TicketTypeFeature
	TicketTypeQuestion
	TicketTypeOther
)

const (
	TicketStatusPending = 1 + iota
	TicketStatusProcessing
	TicketStatusCompleted
)

const (
	TicketEntryTypeMessage = 1 + iota
	TicketEntryTypeStatusChange
)

var ErrTicketNotFound = errors.New("工单不存在")

var ticketTypeToName = map[int]string{
	TicketTypeBug:      "bug",
	TicketTypeFeature:  "feature",
	TicketTypeQuestion: "question",
	TicketTypeOther:    "other",
}

var ticketStatusToName = map[int]string{
	TicketStatusPending:    "pending",
	TicketStatusProcessing: "processing",
	TicketStatusCompleted:  "completed",
}

var ticketNameToType = map[string]int{
	"bug":      TicketTypeBug,
	"feature":  TicketTypeFeature,
	"question": TicketTypeQuestion,
	"other":    TicketTypeOther,
}

var ticketNameToStatus = map[string]int{
	"pending":    TicketStatusPending,
	"processing": TicketStatusProcessing,
	"completed":  TicketStatusCompleted,
}

type Ticket struct {
	Id        int            `json:"id" gorm:"index:idx_ticket_updated_at_id,priority:2;index:idx_ticket_user_updated_id,priority:3;index:idx_ticket_status_updated_id,priority:3"`
	UserId    int            `json:"user_id" gorm:"index;index:idx_ticket_user_updated_id,priority:1"`
	Title     string         `json:"title" gorm:"type:varchar(255);not null"`
	Type      int            `json:"type" gorm:"type:int;default:1"`
	Status    int            `json:"status" gorm:"type:int;default:1;index:idx_ticket_status_updated_id,priority:1"`
	CreatedAt int64          `json:"created_at" gorm:"bigint;index"`
	UpdatedAt int64          `json:"updated_at" gorm:"bigint;index:idx_ticket_updated_at_id,priority:1;index:idx_ticket_user_updated_id,priority:2;index:idx_ticket_status_updated_id,priority:2"`
	ClosedAt  int64          `json:"closed_at" gorm:"bigint;default:0"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type TicketEntry struct {
	Id           int    `json:"id" gorm:"index:idx_ticket_entry_ticket_id_id,priority:2"`
	TicketId     int    `json:"ticket_id" gorm:"index;index:idx_ticket_entry_ticket_id_id,priority:1"`
	EntryType    int    `json:"entry_type" gorm:"type:int;default:1"`
	SenderUserId int    `json:"sender_user_id" gorm:"index"`
	SenderName   string `json:"sender_name" gorm:"type:varchar(64);not null"`
	SenderRole   int    `json:"sender_role" gorm:"type:int;default:1"`
	Content      string `json:"content" gorm:"type:text"`
	FromStatus   int    `json:"from_status" gorm:"type:int;default:0"`
	ToStatus     int    `json:"to_status" gorm:"type:int;default:0"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint;index"`
}

type TicketListFilter struct {
	UserId int
	Status int

	Keyword string
	Offset  int
	Limit   int
}

func TicketTypeName(ticketType int) string {
	if name, ok := ticketTypeToName[ticketType]; ok {
		return name
	}
	return "other"
}

func TicketStatusName(status int) string {
	if name, ok := ticketStatusToName[status]; ok {
		return name
	}
	return "pending"
}

func ParseTicketType(value string) (int, error) {
	ticketType, ok := ticketNameToType[strings.TrimSpace(value)]
	if !ok {
		return 0, errors.New("无效的工单类型")
	}
	return ticketType, nil
}

func ParseTicketStatus(value string) (int, error) {
	status, ok := ticketNameToStatus[strings.TrimSpace(value)]
	if !ok {
		return 0, errors.New("无效的工单状态")
	}
	return status, nil
}

func ListUserTickets(filter TicketListFilter) ([]*Ticket, int64, error) {
	if filter.UserId <= 0 {
		return nil, 0, errors.New("无效的用户")
	}
	query := buildTicketListQuery(DB.Model(&Ticket{}).Where("user_id = ?", filter.UserId), filter)
	return listTickets(query, filter)
}

func ListAdminTickets(filter TicketListFilter) ([]*Ticket, int64, error) {
	query := buildTicketListQuery(DB.Model(&Ticket{}), filter)
	return listTickets(query, filter)
}

func buildTicketListQuery(query *gorm.DB, filter TicketListFilter) *gorm.DB {
	if filter.Status > 0 {
		query = query.Where("status = ?", filter.Status)
	}

	keyword := strings.TrimSpace(filter.Keyword)
	if keyword != "" {
		escaped := escapeTicketLikeValue(keyword)
		pattern := "%" + escaped + "%"
		if id, err := strconv.Atoi(keyword); err == nil {
			query = query.Where("(id = ? OR title LIKE ? ESCAPE '!')", id, pattern)
		} else {
			query = query.Where("title LIKE ? ESCAPE '!'", pattern)
		}
	}

	return query
}

func listTickets(query *gorm.DB, filter TicketListFilter) ([]*Ticket, int64, error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.New("查询工单失败")
	}

	var tickets []*Ticket
	if err := query.Order("updated_at DESC, id DESC").Limit(filter.Limit).Offset(filter.Offset).Find(&tickets).Error; err != nil {
		return nil, 0, errors.New("查询工单失败")
	}
	return tickets, total, nil
}

func GetTicketByID(id int) (*Ticket, error) {
	var ticket Ticket
	if err := DB.First(&ticket, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	return &ticket, nil
}

func GetTicketByIDForUpdate(tx *gorm.DB, id int) (*Ticket, error) {
	var ticket Ticket
	if err := tx.First(&ticket, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	return &ticket, nil
}

func GetTicketEntries(ticketId int) ([]*TicketEntry, error) {
	var entries []*TicketEntry
	if err := DB.Where("ticket_id = ?", ticketId).Order("id ASC").Find(&entries).Error; err != nil {
		return nil, errors.New("查询工单详情失败")
	}
	return entries, nil
}

func CreateTicketTx(tx *gorm.DB, ticket *Ticket) error {
	return tx.Create(ticket).Error
}

func CreateTicketEntryTx(tx *gorm.DB, entry *TicketEntry) error {
	return tx.Create(entry).Error
}

func UpdateTicketFieldsTx(tx *gorm.DB, ticketId int, values map[string]any) error {
	return tx.Model(&Ticket{}).Where("id = ?", ticketId).Updates(values).Error
}

func UpdateTicketStatusTx(tx *gorm.DB, ticketId int, fromStatus int, values map[string]any) (bool, error) {
	result := tx.Model(&Ticket{}).Where("id = ? AND status = ?", ticketId, fromStatus).Updates(values)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func escapeTicketLikeValue(value string) string {
	value = strings.ReplaceAll(value, "!", "!!")
	value = strings.ReplaceAll(value, "%", "!%")
	value = strings.ReplaceAll(value, "_", "!_")
	return value
}

func NewTicket(title string, userId int, ticketType int) *Ticket {
	now := common.GetTimestamp()
	return &Ticket{
		UserId:    userId,
		Title:     title,
		Type:      ticketType,
		Status:    TicketStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
