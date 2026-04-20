package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
)

// UserRepository предоставляет методы для работы с пользователями в БД.
type UserRepository struct {
	db *database.DB
}

// NewUserRepository создает новый экземпляр UserRepository.
func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

// userSelectBase — базовый SELECT для получения пользователя с department.
const userSelectBase = `
	SELECT u.id, u.login, u.password_hash, u.full_name, u.is_document_participant, u.is_active, u.failed_login_attempts, u.created_at, u.updated_at,
	       d.id, d.name
	FROM users u
	LEFT JOIN departments d ON u.department_id = d.id`

// getUserByCondition — общий метод для получения пользователя по произвольному условию WHERE.
func (r *UserRepository) getUserByCondition(whereClause string, arg interface{}) (*models.User, error) {
	user := &models.User{}

	var departmentID sql.NullString
	var departmentName sql.NullString

	query := userSelectBase + " " + whereClause
	err := r.db.QueryRow(query, arg).Scan(
		&user.ID, &user.Login, &user.PasswordHash, &user.FullName,
		&user.IsDocumentParticipant, &user.IsActive, &user.FailedLoginAttempts, &user.CreatedAt, &user.UpdatedAt,
		&departmentID, &departmentName,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if departmentID.Valid {
		uid, _ := uuid.Parse(departmentID.String)
		user.DepartmentID = &uid
		user.Department = &models.Department{
			ID:   uid,
			Name: departmentName.String,
		}
		if noms, err := r.getDepartmentNomenclatureIDs(uid); err == nil {
			user.Department.NomenclatureIDs = noms
		}
	}

	systemPermissions, err := r.GetUserSystemPermissions(user.ID)
	if err != nil {
		return nil, err
	}
	user.SystemPermissions = systemPermissions

	return user, nil
}

// GetByLogin возвращает пользователя по его логину.
func (r *UserRepository) GetByLogin(login string) (*models.User, error) {
	return r.getUserByCondition("WHERE u.login = $1", login)
}

// GetByID возвращает пользователя по его ID.
func (r *UserRepository) GetByID(id uuid.UUID) (*models.User, error) {
	return r.getUserByCondition("WHERE u.id = $1", id)
}

// GetAll возвращает список всех пользователей.
func (r *UserRepository) GetAll() ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.login, u.full_name, u.is_document_participant, u.is_active, u.failed_login_attempts, u.created_at, u.updated_at,
		       d.id, d.name
		FROM users u
		LEFT JOIN departments d ON u.department_id = d.id
		ORDER BY u.full_name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}
	defer rows.Close()

	users := make([]models.User, 0)
	userIDs := make([]uuid.UUID, 0)
	departmentIDs := make([]uuid.UUID, 0)
	departmentIndexes := make(map[uuid.UUID][]int) // departmentID -> indexes of users with this department

	for rows.Next() {
		var user models.User
		var departmentID sql.NullString
		var departmentName sql.NullString

		if err := rows.Scan(
			&user.ID, &user.Login, &user.FullName,
			&user.IsDocumentParticipant, &user.IsActive, &user.FailedLoginAttempts, &user.CreatedAt, &user.UpdatedAt,
			&departmentID, &departmentName,
		); err != nil {
			return nil, err
		}

		if departmentID.Valid {
			uid, _ := uuid.Parse(departmentID.String)
			user.DepartmentID = &uid
			user.Department = &models.Department{
				ID:   uid,
				Name: departmentName.String,
			}
			if _, exists := departmentIndexes[uid]; !exists {
				departmentIDs = append(departmentIDs, uid)
			}
			departmentIndexes[uid] = append(departmentIndexes[uid], len(users))
		}

		userIDs = append(userIDs, user.ID)
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return users, nil
	}

	if err := r.batchLoadUserSystemPermissions(users, userIDs); err != nil {
		return nil, err
	}

	// Batch-загрузка номенклатур подразделений одним запросом
	if err := r.batchLoadDepartmentNomenclatures(users, departmentIDs, departmentIndexes); err != nil {
		return nil, err
	}

	return users, nil
}

// Create создает нового пользователя в БД.
func (r *UserRepository) Create(req models.CreateUserRequest) (*models.User, error) {
	if err := security.ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	passwordHash, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	var depID *uuid.UUID
	if req.DepartmentID != "" {
		if uid, err := uuid.Parse(req.DepartmentID); err == nil {
			depID = &uid
		}
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var userID uuid.UUID
	err = tx.QueryRow(`
		INSERT INTO users (login, password_hash, full_name, department_id, is_document_participant)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, req.Login, passwordHash, req.FullName, depID, req.IsDocumentParticipant).Scan(&userID)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetByID(userID)
}

// Update обновляет данные существующего пользователя.
func (r *UserRepository) Update(req models.UpdateUserRequest) (*models.User, error) {
	uid, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	var depID *uuid.UUID
	if req.DepartmentID != "" {
		if uid, err := uuid.Parse(req.DepartmentID); err == nil {
			depID = &uid
		}
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE users
		SET login = $1,
		    full_name = $2,
		    is_active = $3,
		    department_id = $4,
		    is_document_participant = $5,
		    failed_login_attempts = CASE
		        WHEN is_active = false AND $3 = true THEN 0
		        ELSE failed_login_attempts
		    END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $6
	`, req.Login, req.FullName, req.IsActive, depID, req.IsDocumentParticipant, uid)

	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetByID(uid)
}

// GetUserSystemPermissions возвращает список системных прав пользователя по его ID.
func (r *UserRepository) GetUserSystemPermissions(userID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT permission FROM user_system_permissions WHERE user_id = $1 AND is_allowed = true
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user system permissions: %w", err)
	}
	defer rows.Close()

	permissions := make([]string, 0)
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}

// GetExecutors возвращает список активных пользователей, доступных для назначения и ознакомления.
func (r *UserRepository) GetExecutors() ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.login, u.full_name, u.is_document_participant, u.is_active, u.created_at, u.updated_at,
		       d.id, d.name
		FROM users u
		LEFT JOIN departments d ON u.department_id = d.id
		WHERE u.is_active = true AND u.is_document_participant = true
		ORDER BY u.full_name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignable users: %w", err)
	}
	defer rows.Close()

	users := make([]models.User, 0)
	userIDs := make([]uuid.UUID, 0)
	departmentIDs := make([]uuid.UUID, 0)
	departmentIndexes := make(map[uuid.UUID][]int)

	for rows.Next() {
		var user models.User
		var departmentID sql.NullString
		var departmentName sql.NullString

		if err := rows.Scan(
			&user.ID, &user.Login, &user.FullName,
			&user.IsDocumentParticipant, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
			&departmentID, &departmentName,
		); err != nil {
			return nil, err
		}

		if departmentID.Valid {
			uid, _ := uuid.Parse(departmentID.String)
			user.DepartmentID = &uid
			user.Department = &models.Department{
				ID:   uid,
				Name: departmentName.String,
			}
			if _, exists := departmentIndexes[uid]; !exists {
				departmentIDs = append(departmentIDs, uid)
			}
			departmentIndexes[uid] = append(departmentIndexes[uid], len(users))
		}

		userIDs = append(userIDs, user.ID)
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return users, nil
	}

	if err := r.batchLoadUserSystemPermissions(users, userIDs); err != nil {
		return nil, err
	}

	// Batch-загрузка номенклатур подразделений одним запросом
	if err := r.batchLoadDepartmentNomenclatures(users, departmentIDs, departmentIndexes); err != nil {
		return nil, err
	}

	return users, nil
}

// UpdatePassword обновляет хэш пароля пользователя.
func (r *UserRepository) UpdatePassword(userID uuid.UUID, newPasswordHash string) error {
	_, err := r.db.Exec(`
		UPDATE users
		SET password_hash = $1,
		    failed_login_attempts = 0,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, newPasswordHash, userID)

	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// ResetPassword сбрасывает (изменяет) пароль пользователя.
func (r *UserRepository) ResetPassword(userID uuid.UUID, newPassword string) error {
	if err := security.ValidatePassword(newPassword); err != nil {
		return err
	}
	hash, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return r.UpdatePassword(userID, hash)
}

// UpdateProfile обновляет данные профиля пользователя (логин, ФИО).
func (r *UserRepository) UpdateProfile(userID uuid.UUID, req models.UpdateProfileRequest) error {
	_, err := r.db.Exec(`
		UPDATE users SET login = $1, full_name = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`, req.Login, req.FullName, userID)

	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	return nil
}

// IncrementFailedLoginAttempts увеличивает счетчик неудачных входов и деактивирует пользователя после 5-й ошибки.
func (r *UserRepository) IncrementFailedLoginAttempts(userID uuid.UUID) (int, bool, error) {
	var attempts int
	var isActive bool

	err := r.db.QueryRow(`
		UPDATE users
		SET failed_login_attempts = failed_login_attempts + 1,
		    is_active = CASE
		        WHEN failed_login_attempts + 1 >= 5 THEN false
		        ELSE is_active
		    END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING failed_login_attempts, is_active
	`, userID).Scan(&attempts, &isActive)
	if err != nil {
		return 0, false, fmt.Errorf("failed to increment failed login attempts: %w", err)
	}

	return attempts, isActive, nil
}

// ResetFailedLoginAttempts сбрасывает счетчик неудачных входов пользователя.
func (r *UserRepository) ResetFailedLoginAttempts(userID uuid.UUID) error {
	_, err := r.db.Exec(`
		UPDATE users
		SET failed_login_attempts = 0,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to reset failed login attempts: %w", err)
	}

	return nil
}

// batchLoadUserSystemPermissions загружает системные права пользователей одним SQL-запросом.
func (r *UserRepository) batchLoadUserSystemPermissions(users []models.User, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}

	userIndex := make(map[uuid.UUID]int, len(userIDs))
	for i, uid := range userIDs {
		userIndex[uid] = i
	}

	rows, err := r.db.Query(`
		SELECT user_id, permission
		FROM user_system_permissions
		WHERE user_id = ANY($1) AND is_allowed = true
	`, pq.Array(userIDs))
	if err != nil {
		return fmt.Errorf("failed to batch load user system permissions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID uuid.UUID
		var permission string
		if err := rows.Scan(&userID, &permission); err != nil {
			return err
		}
		if idx, ok := userIndex[userID]; ok {
			users[idx].SystemPermissions = append(users[idx].SystemPermissions, permission)
		}
	}
	return rows.Err()
}

// batchLoadDepartmentNomenclatures загружает номенклатуры подразделений одним SQL-запросом
// вместо N отдельных запросов (решение проблемы N+1).
func (r *UserRepository) batchLoadDepartmentNomenclatures(users []models.User, departmentIDs []uuid.UUID, departmentIndexes map[uuid.UUID][]int) error {
	if len(departmentIDs) == 0 {
		return nil
	}

	rows, err := r.db.Query(`
		SELECT department_id, nomenclature_id
		FROM department_nomenclature
		WHERE department_id = ANY($1)
	`, pq.Array(departmentIDs))
	if err != nil {
		return fmt.Errorf("failed to batch load department nomenclatures: %w", err)
	}
	defer rows.Close()

	// Собираем номенклатуры по departmentID
	nomMap := make(map[uuid.UUID][]string)
	for rows.Next() {
		var depID uuid.UUID
		var nomID uuid.UUID
		if err := rows.Scan(&depID, &nomID); err != nil {
			return err
		}
		nomMap[depID] = append(nomMap[depID], nomID.String())
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Присваиваем номенклатуры пользователям
	for depID, noms := range nomMap {
		for _, idx := range departmentIndexes[depID] {
			if users[idx].Department != nil {
				users[idx].Department.NomenclatureIDs = noms
			}
		}
	}

	return nil
}

func (r *UserRepository) getDepartmentNomenclatureIDs(departmentID uuid.UUID) ([]string, error) {
	query := `
		SELECT nomenclature_id
		FROM department_nomenclature
		WHERE department_id = $1
	`
	rows, err := r.db.Query(query, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id.String())
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

// CountUsers возвращает общее количество пользователей в базе данных.
func (r *UserRepository) CountUsers() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}
