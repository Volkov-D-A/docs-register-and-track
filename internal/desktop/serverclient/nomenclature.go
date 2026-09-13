package serverclient

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type NomenclatureClient interface {
	ListNomenclature(context.Context, int, string) ([]dto.Nomenclature, error)
	ListActiveNomenclature(context.Context, string) ([]dto.Nomenclature, error)
	CreateNomenclature(context.Context, string, string, int, string, string, string, int) (*dto.Nomenclature, error)
	UpdateNomenclature(context.Context, string, string, string, int, string, string, string, bool) (*dto.Nomenclature, error)
	DeleteNomenclature(context.Context, string) error
}

type nomenclatureCreateRequest struct {
	Name          string `json:"name"`
	Index         string `json:"index"`
	Year          int    `json:"year"`
	KindCode      string `json:"kindCode"`
	Separator     string `json:"separator"`
	NumberingMode string `json:"numberingMode"`
	StartNumber   int    `json:"startNumber"`
}

type nomenclatureUpdateRequest struct {
	Name          string `json:"name"`
	Index         string `json:"index"`
	Year          int    `json:"year"`
	KindCode      string `json:"kindCode"`
	Separator     string `json:"separator"`
	NumberingMode string `json:"numberingMode"`
	IsActive      bool   `json:"isActive"`
}

func (c *Client) ListNomenclature(ctx context.Context, year int, kindCode string) ([]dto.Nomenclature, error) {
	values := url.Values{}
	if year > 0 {
		values.Set("year", strconv.Itoa(year))
	}
	if kindCode != "" {
		values.Set("kindCode", kindCode)
	}
	path := "/api/v1/nomenclature"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var result []dto.Nomenclature
	if err := c.doUserRequest(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ListActiveNomenclature(ctx context.Context, kindCode string) ([]dto.Nomenclature, error) {
	values := url.Values{}
	values.Set("kindCode", kindCode)
	var result []dto.Nomenclature
	if err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/nomenclature/active?"+values.Encode(), nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) CreateNomenclature(ctx context.Context, name, index string, year int, kindCode, separator, numberingMode string, startNumber int) (*dto.Nomenclature, error) {
	body := nomenclatureCreateRequest{name, index, year, kindCode, separator, numberingMode, startNumber}
	var result dto.Nomenclature
	if err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/nomenclature", body, http.StatusCreated, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateNomenclature(ctx context.Context, id, name, index string, year int, kindCode, separator, numberingMode string, isActive bool) (*dto.Nomenclature, error) {
	path, err := nomenclatureItemPath(id)
	if err != nil {
		return nil, err
	}
	body := nomenclatureUpdateRequest{name, index, year, kindCode, separator, numberingMode, isActive}
	var result dto.Nomenclature
	if err := c.doUserRequest(ctx, http.MethodPatch, path, body, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteNomenclature(ctx context.Context, id string) error {
	path, err := nomenclatureItemPath(id)
	if err != nil {
		return err
	}
	return c.doUserRequest(ctx, http.MethodDelete, path, nil, http.StatusNoContent, nil)
}

func nomenclatureItemPath(id string) (string, error) {
	if _, err := uuid.Parse(id); err != nil {
		return "", models.NewBadRequestWrapped("неверный ID номенклатуры", err)
	}
	return "/api/v1/nomenclature/" + url.PathEscape(id), nil
}
