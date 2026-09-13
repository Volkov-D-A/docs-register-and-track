package serverclient

import (
	"context"
	"net/http"
	"net/url"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type AssignmentClient interface {
	CreateAssignment(context.Context, string, string, string, string, []string) (*dto.Assignment, error)
	CreateAssignmentSeries(context.Context, models.AssignmentSeriesRequest) (*dto.AssignmentSeries, error)
	GetAssignmentSeries(context.Context, string) (*dto.AssignmentSeries, error)
	GetAssignmentSeriesHistory(context.Context, string) ([]dto.Assignment, error)
	UpdateAssignmentSeries(context.Context, string, models.AssignmentSeriesRequest) (*dto.AssignmentSeries, error)
	CancelAssignmentSeries(context.Context, string) error
	UpdateAssignment(context.Context, string, string, string, string, []string) (*dto.Assignment, error)
	UpdateAssignmentStatus(context.Context, string, string, string) (*dto.Assignment, error)
	GetAssignment(context.Context, string) (*dto.Assignment, error)
	ListAssignments(context.Context, models.AssignmentFilter) (*dto.PagedResult[dto.Assignment], error)
	DeleteAssignment(context.Context, string) error
}

type assignmentDetailsRequest struct {
	DocumentID    string   `json:"documentId,omitempty"`
	ExecutorID    string   `json:"executorId"`
	Content       string   `json:"content"`
	Deadline      string   `json:"deadline"`
	CoExecutorIDs []string `json:"coExecutorIds"`
}

type assignmentStatusRequest struct {
	Status string `json:"status"`
	Report string `json:"report"`
}

func (c *Client) CreateAssignment(ctx context.Context, documentID, executorID, content, deadline string, coExecutorIDs []string) (*dto.Assignment, error) {
	request := assignmentDetailsRequest{DocumentID: documentID, ExecutorID: executorID, Content: content, Deadline: deadline, CoExecutorIDs: coExecutorIDs}
	var result dto.Assignment
	if err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/assignments", request, http.StatusCreated, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetAssignment(ctx context.Context, id string) (*dto.Assignment, error) {
	var result dto.Assignment
	if err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/assignments/"+url.PathEscape(id), nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ListAssignments(ctx context.Context, filter models.AssignmentFilter) (*dto.PagedResult[dto.Assignment], error) {
	filter.AllowedDocumentKinds = nil
	filter.AccessibleByUserID = ""
	filter.AccessibleByUserIDs = nil
	var result dto.PagedResult[dto.Assignment]
	if err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/assignments/query", filter, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateAssignment(ctx context.Context, id, executorID, content, deadline string, coExecutorIDs []string) (*dto.Assignment, error) {
	request := assignmentDetailsRequest{ExecutorID: executorID, Content: content, Deadline: deadline, CoExecutorIDs: coExecutorIDs}
	var result dto.Assignment
	if err := c.doUserRequest(ctx, http.MethodPatch, "/api/v1/assignments/"+url.PathEscape(id), request, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateAssignmentStatus(ctx context.Context, id, status, report string) (*dto.Assignment, error) {
	var result dto.Assignment
	if err := c.doUserRequest(ctx, http.MethodPatch, "/api/v1/assignments/"+url.PathEscape(id)+"/status", assignmentStatusRequest{Status: status, Report: report}, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteAssignment(ctx context.Context, id string) error {
	return c.doUserRequest(ctx, http.MethodDelete, "/api/v1/assignments/"+url.PathEscape(id), nil, http.StatusNoContent, nil)
}

func (c *Client) CreateAssignmentSeries(ctx context.Context, request models.AssignmentSeriesRequest) (*dto.AssignmentSeries, error) {
	var result dto.AssignmentSeries
	if err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/assignment-series", request, http.StatusCreated, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetAssignmentSeries(ctx context.Context, id string) (*dto.AssignmentSeries, error) {
	var result dto.AssignmentSeries
	if err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/assignment-series/"+url.PathEscape(id), nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetAssignmentSeriesHistory(ctx context.Context, id string) ([]dto.Assignment, error) {
	var result []dto.Assignment
	if err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/assignment-series/"+url.PathEscape(id)+"/history", nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateAssignmentSeries(ctx context.Context, id string, request models.AssignmentSeriesRequest) (*dto.AssignmentSeries, error) {
	var result dto.AssignmentSeries
	if err := c.doUserRequest(ctx, http.MethodPatch, "/api/v1/assignment-series/"+url.PathEscape(id), request, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) CancelAssignmentSeries(ctx context.Context, id string) error {
	return c.doUserRequest(ctx, http.MethodDelete, "/api/v1/assignment-series/"+url.PathEscape(id), nil, http.StatusNoContent, nil)
}
