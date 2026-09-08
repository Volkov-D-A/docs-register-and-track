package services

import (
	"context"
	"sync"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

// SystemService предоставляет проверку совместимости и готовности сервера для фронтенда.
type SystemService struct {
	ctx           context.Context
	ctxMu         sync.RWMutex
	client        serverclient.SystemClient
	clientVersion string
}

// NewSystemService returns the desktop adapter and its Wails startup callback.
func NewSystemService(client serverclient.SystemClient, clientVersion string) (*SystemService, func(context.Context)) {
	s := &SystemService{client: client, clientVersion: clientVersion}
	return s, func(ctx context.Context) {
		s.ctxMu.Lock()
		defer s.ctxMu.Unlock()
		s.ctx = ctx
	}
}

// GetBootstrapStatus checks compatibility and readiness before login is shown.
func (s *SystemService) GetBootstrapStatus() *dto.BootstrapStatus {
	if s.client == nil || s.clientVersion == "" {
		return bootstrapFailure(serverclient.SystemErrorProtocol)
	}
	s.ctxMu.RLock()
	parent := s.ctx
	s.ctxMu.RUnlock()
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()

	compatibility, err := s.client.Compatibility(ctx, s.clientVersion)
	if err != nil {
		return bootstrapFailure(serverclient.SystemRequestErrorKind(err))
	}
	if !compatibility.Compatible {
		message := "Версия приложения несовместима с сервером. Установите актуальную версию приложения."
		if compatibility.Code == "client_too_new" {
			message = "Версия приложения новее версии сервера. Сначала обновите сервер."
		}
		return &dto.BootstrapStatus{State: compatibility.Code, Code: compatibility.Code, Message: message, Compatibility: compatibility}
	}
	status, err := s.client.SystemStatus(ctx)
	if err != nil {
		return bootstrapFailure(serverclient.SystemRequestErrorKind(err))
	}
	result := &dto.BootstrapStatus{State: status.Status, Code: status.Code, Compatibility: compatibility, System: status}
	switch status.Status {
	case "ready":
		result.Message = "Сервер готов к работе."
	case "maintenance":
		result.Message = "На сервере выполняется обслуживание. Часть операций может быть временно недоступна."
	default:
		result.State = "not_ready"
		result.Message = "Сервер пока не готов к работе. Повторите попытку позже."
	}
	return result
}

func bootstrapFailure(code string) *dto.BootstrapStatus {
	message := "Сервер вернул некорректный ответ. Обратитесь к администратору."
	switch code {
	case serverclient.SystemErrorTLS:
		message = "Не удалось проверить сертификат сервера. Обратитесь к администратору."
	case serverclient.SystemErrorUnavailable:
		message = "Сервер недоступен. Проверьте подключение и повторите попытку."
	}
	return &dto.BootstrapStatus{State: code, Code: code, Message: message}
}
