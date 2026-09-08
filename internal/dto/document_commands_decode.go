package dto

import (
	"bytes"
	"encoding/json"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// NormalizeDocumentRegisterRequest decodes the transport shape only; permissions
// and business validation remain the responsibility of server command handlers.
func NormalizeDocumentRegisterRequest(kind models.DocumentKind, req any) (any, error) {
	switch kind {
	case models.DocumentKindIncomingLetter:
		if typedReq, ok := req.(IncomingLetterRegisterRequest); ok {
			return typedReq, nil
		}

		var typedReq IncomingLetterRegisterRequest
		if err := decodeDocumentCommandRequest(req, &typedReq); err != nil {
			return nil, err
		}

		return typedReq, nil
	case models.DocumentKindOutgoingLetter:
		if typedReq, ok := req.(OutgoingLetterRegisterRequest); ok {
			return typedReq, nil
		}

		var typedReq OutgoingLetterRegisterRequest
		if err := decodeDocumentCommandRequest(req, &typedReq); err != nil {
			return nil, err
		}

		return typedReq, nil
	case models.DocumentKindCitizenAppeal:
		if typedReq, ok := req.(CitizenAppealRegisterRequest); ok {
			return typedReq, nil
		}

		var typedReq CitizenAppealRegisterRequest
		if err := decodeDocumentCommandRequest(req, &typedReq); err != nil {
			return nil, err
		}

		return typedReq, nil
	case models.DocumentKindAdministrativeOrder:
		if typedReq, ok := req.(AdministrativeOrderRegisterRequest); ok {
			return typedReq, nil
		}

		var typedReq AdministrativeOrderRegisterRequest
		if err := decodeDocumentCommandRequest(req, &typedReq); err != nil {
			return nil, err
		}

		return typedReq, nil
	default:
		return nil, models.ErrForbidden
	}
}

// NormalizeDocumentUpdateRequest decodes the transport shape without authorizing
// or validating the business operation.
func NormalizeDocumentUpdateRequest(kind models.DocumentKind, req any) (any, error) {
	switch kind {
	case models.DocumentKindIncomingLetter:
		if typedReq, ok := req.(IncomingLetterUpdateRequest); ok {
			return typedReq, nil
		}

		var typedReq IncomingLetterUpdateRequest
		if err := decodeDocumentCommandRequest(req, &typedReq); err != nil {
			return nil, err
		}

		return typedReq, nil
	case models.DocumentKindOutgoingLetter:
		if typedReq, ok := req.(OutgoingLetterUpdateRequest); ok {
			return typedReq, nil
		}

		var typedReq OutgoingLetterUpdateRequest
		if err := decodeDocumentCommandRequest(req, &typedReq); err != nil {
			return nil, err
		}

		return typedReq, nil
	case models.DocumentKindCitizenAppeal:
		if typedReq, ok := req.(CitizenAppealUpdateRequest); ok {
			return typedReq, nil
		}

		var typedReq CitizenAppealUpdateRequest
		if err := decodeDocumentCommandRequest(req, &typedReq); err != nil {
			return nil, err
		}

		return typedReq, nil
	case models.DocumentKindAdministrativeOrder:
		if typedReq, ok := req.(AdministrativeOrderUpdateRequest); ok {
			return typedReq, nil
		}

		var typedReq AdministrativeOrderUpdateRequest
		if err := decodeDocumentCommandRequest(req, &typedReq); err != nil {
			return nil, err
		}

		return typedReq, nil
	default:
		return nil, models.ErrForbidden
	}
}

func decodeDocumentCommandRequest(src any, dst any) error {
	data, err := json.Marshal(src)
	if err != nil {
		return models.NewBadRequest("неверный формат команды документа")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return models.NewBadRequest("неверные поля команды документа")
	}

	return nil
}
