package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
)

const systemAPIVersion = "v1"

func (api *managementAPI) systemStatus(w http.ResponseWriter, r *http.Request) {
	result := dto.SystemStatus{
		Status:        "not_ready",
		Code:          "status_unavailable",
		APIVersion:    systemAPIVersion,
		ServerVersion: api.serverVersion,
	}
	status, err := api.migrations.GetMigrationStatus(database.DefaultMigrationsPath)
	if err != nil || status == nil {
		writeJSON(w, http.StatusOK, result)
		return
	}
	result.Schema = dto.SystemSchemaStatus{
		CurrentVersion:  status.CurrentVersion,
		RequiredVersion: status.LatestAvailableVersion,
		Compatible:      status.Compatible,
		Dirty:           status.Dirty,
	}
	if err := api.lifecycle.CheckReady(); err != nil {
		result.Status = "maintenance"
		result.Code = "maintenance"
		result.Maintenance = true
		writeJSON(w, http.StatusOK, result)
		return
	}
	if !status.UpToDate || !status.Compatible || status.Dirty {
		result.Code = "schema_not_ready"
		writeJSON(w, http.StatusOK, result)
		return
	}
	if api.readinessCheck != nil {
		if err := api.readinessCheck(r.Context(), api.cfg); err != nil {
			result.Code = "dependency_not_ready"
			writeJSON(w, http.StatusOK, result)
			return
		}
	}
	result.Status = "ready"
	result.Code = "ready"
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) systemCompatibility(w http.ResponseWriter, r *http.Request) {
	clientVersion := strings.TrimSpace(r.URL.Query().Get("clientVersion"))
	client, err := parseSemanticVersion(clientVersion)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_client_version", err)
		return
	}
	server, err := parseSemanticVersion(api.serverVersion)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "invalid_server_version", err)
		return
	}
	code := "compatible"
	compatible := client == server
	if client.less(server) {
		code = "client_too_old"
	} else if server.less(client) {
		code = "client_too_new"
	}
	writeJSON(w, http.StatusOK, dto.CompatibilityResult{
		Compatible:           compatible,
		Code:                 code,
		APIVersion:           systemAPIVersion,
		ServerVersion:        api.serverVersion,
		MinimumClientVersion: api.serverVersion,
		MaximumClientVersion: api.serverVersion,
	})
}

type semanticVersion [3]uint64

func parseSemanticVersion(raw string) (semanticVersion, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return semanticVersion{}, fmt.Errorf("version must have major.minor.patch format")
	}
	var version semanticVersion
	for index, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return semanticVersion{}, fmt.Errorf("version must have major.minor.patch format")
		}
		for _, character := range part {
			if character < '0' || character > '9' {
				return semanticVersion{}, fmt.Errorf("version must have major.minor.patch format")
			}
		}
		value, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return semanticVersion{}, fmt.Errorf("version must have major.minor.patch format")
		}
		version[index] = value
	}
	return version, nil
}

func (v semanticVersion) less(other semanticVersion) bool {
	for index := range v {
		if v[index] != other[index] {
			return v[index] < other[index]
		}
	}
	return false
}
