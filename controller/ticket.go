package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/service"
)

type createTicketRequest struct {
	Title   string `json:"title"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

type replyTicketRequest struct {
	Content string `json:"content"`
}

type updateTicketStatusRequest struct {
	Status string `json:"status"`
}

func GetUserTickets(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := service.ListUserTickets(c.GetInt("id"), pageInfo.GetPage(), pageInfo.GetPageSize(), c.DefaultQuery("status", "all"), c.Query("keyword"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetAdminTickets(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := service.ListAdminTickets(c.GetInt("role"), pageInfo.GetPage(), pageInfo.GetPageSize(), c.DefaultQuery("status", "all"), c.Query("keyword"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func CreateTicket(c *gin.Context) {
	var req createTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	data, err := service.CreateTicket(service.CreateTicketInput{
		UserId:   c.GetInt("id"),
		Username: c.GetString("username"),
		Role:     c.GetInt("role"),
		Title:    req.Title,
		Type:     req.Type,
		Content:  req.Content,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func GetTicketDetail(c *gin.Context) {
	ticketId, err := strconv.Atoi(c.Param("id"))
	if err != nil || ticketId <= 0 {
		common.ApiErrorMsg(c, "无效的工单编号")
		return
	}

	data, err := service.GetTicketDetail(ticketId, c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func ReplyTicket(c *gin.Context) {
	ticketId, err := strconv.Atoi(c.Param("id"))
	if err != nil || ticketId <= 0 {
		common.ApiErrorMsg(c, "无效的工单编号")
		return
	}

	var req replyTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	err = service.ReplyTicket(service.ReplyTicketInput{
		TicketId: ticketId,
		UserId:   c.GetInt("id"),
		Username: c.GetString("username"),
		Role:     c.GetInt("role"),
		Content:  req.Content,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func CloseTicket(c *gin.Context) {
	ticketId, err := strconv.Atoi(c.Param("id"))
	if err != nil || ticketId <= 0 {
		common.ApiErrorMsg(c, "无效的工单编号")
		return
	}

	err = service.CloseTicket(ticketId, c.GetInt("id"), c.GetInt("role"), c.GetString("username"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func UpdateTicketStatus(c *gin.Context) {
	ticketId, err := strconv.Atoi(c.Param("id"))
	if err != nil || ticketId <= 0 {
		common.ApiErrorMsg(c, "无效的工单编号")
		return
	}

	var req updateTicketStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	err = service.UpdateTicketStatus(ticketId, c.GetInt("id"), c.GetInt("role"), c.GetString("username"), req.Status)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
