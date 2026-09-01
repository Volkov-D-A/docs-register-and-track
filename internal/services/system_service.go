package services

import (
	"context"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

// SystemService предоставляет системные методы для фронтенда (проверка БД и др.).
type SystemService struct {
	ctx           context.Context
	client        serverclient.SystemClient
	clientVersion string
}

// NewSystemService создает новый экземпляр SystemService.
func NewSystemService() *SystemService { return &SystemService{} }

// NewSystemServiceWithClient creates the runtime service used by the desktop app.
func NewSystemServiceWithClient(client serverclient.SystemClient, clientVersion string) *SystemService {
	return &SystemService{client: client, clientVersion: clientVersion}
}

// Startup вызывается Wails при старте приложения
func (s *SystemService) Startup(ctx context.Context) {
	s.ctx = ctx
}

// GetBootstrapStatus checks compatibility and readiness before login is shown.
func (s *SystemService) GetBootstrapStatus() *dto.BootstrapStatus {
	if s.client == nil || s.clientVersion == "" {
		return bootstrapFailure(serverclient.SystemErrorProtocol)
	}
	parent := s.ctx
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
