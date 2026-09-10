package backup

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/Volkov-D-A/docs-register-and-track/internal/backup/smb"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type Settings models.BackupSettings
type SettingsUpdate struct {
	Settings      Settings `json:"settings"`
	Password      string   `json:"password"`
	ClearPassword bool     `json:"clearPassword"`
}
type StoredSettings struct {
	Settings Settings        `json:"settings"`
	Secret   EncryptedSecret `json:"secret"`
}

func DefaultSettings() Settings {
	return Settings{Time: "02:00", Timezone: "Asia/Yekaterinburg", Weekdays: []int{0, 1, 2, 3, 4, 5, 6}, RetentionDays: 15, KeepCopies: 3}
}
func (s Settings) Validate() error {
	if err := smb.Config(s.SMB).Validate(); err != nil {
		return err
	}
	if _, err := time.LoadLocation(s.Timezone); err != nil {
		return fmt.Errorf("invalid timezone")
	}
	if _, err := time.Parse("15:04", s.Time); err != nil {
		return fmt.Errorf("backup time must be HH:MM")
	}
	if len(s.Weekdays) == 0 || len(s.Weekdays) > 7 {
		return fmt.Errorf("choose backup weekdays")
	}
	seen := map[int]bool{}
	for _, day := range s.Weekdays {
		if day < 0 || day > 6 || seen[day] {
			return fmt.Errorf("invalid backup weekdays")
		}
		seen[day] = true
	}
	if s.RetentionDays < 1 || s.RetentionDays > 3650 || s.KeepCopies < 1 || s.KeepCopies > 1000 {
		return fmt.Errorf("invalid retention settings")
	}
	return nil
}

// Due returns today's scheduled occurrence, including one missed occurrence after downtime.
// time.Date resolves daylight-saving transitions; a date is scheduled at most once.
func (s Settings) Due(now time.Time) (time.Time, bool) {
	if !s.Enabled {
		return time.Time{}, false
	}
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		return time.Time{}, false
	}
	local := now.In(loc)
	parts := strings.Split(s.Time, ":")
	if len(parts) != 2 {
		return time.Time{}, false
	}
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])
	for days := 0; days < 8; days++ {
		date := local.AddDate(0, 0, -days)
		eligible := false
		for _, day := range s.Weekdays {
			if int(date.Weekday()) == day {
				eligible = true
			}
		}
		if !eligible {
			continue
		}
		due := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, loc)
		if !due.After(now) {
			return due, true
		}
	}
	return time.Time{}, false
}

type SettingsRepository struct{ DB *sql.DB }

func (r SettingsRepository) Load(ctx context.Context) (StoredSettings, error) {
	result := StoredSettings{Settings: DefaultSettings()}
	var raw []byte
	err := r.DB.QueryRowContext(ctx, "SELECT settings FROM backup_settings WHERE id=true").Scan(&raw)
	if err == sql.ErrNoRows {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(raw, &result)
	result.Settings.PasswordSet = result.Secret.Data != ""
	return result, err
}
func (r SettingsRepository) Save(ctx context.Context, value StoredSettings, actor string) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO backup_settings(id,settings) VALUES(true,$1) ON CONFLICT(id) DO UPDATE SET settings=excluded.settings`, raw); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO backup_audit(actor,action) VALUES($1,'settings_updated')`, actor); err != nil {
		return err
	}
	return tx.Commit()
}

func (s Settings) Next(now time.Time) time.Time {
	if !s.Enabled {
		return time.Time{}
	}
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		return time.Time{}
	}
	local := now.In(loc)
	clock, err := time.Parse("15:04", s.Time)
	if err != nil {
		return time.Time{}
	}
	for offset := 0; offset < 8; offset++ {
		date := local.AddDate(0, 0, offset)
		for _, day := range s.Weekdays {
			if int(date.Weekday()) != day {
				continue
			}
			candidate := time.Date(date.Year(), date.Month(), date.Day(), clock.Hour(), clock.Minute(), 0, 0, loc)
			if candidate.After(now) {
				return candidate
			}
		}
	}
	return time.Time{}
}
