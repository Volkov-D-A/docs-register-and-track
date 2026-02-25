package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"docflow/internal/database"
	"docflow/internal/models"
	"docflow/internal/security"
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
	SELECT u.id, u.login, u.password_hash, u.full_name, u.is_active, u.created_at, u.updated_at,
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
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
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

	roles, err := r.GetUserRoles(user.ID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

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
		SELECT u.id, u.login, u.full_name, u.is_active, u.created_at, u.updated_at,
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
			&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
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

	// Batch-загрузка ролей для всех пользователей одним запросом
	if err := r.batchLoadUserRoles(users, userIDs); err != nil {
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
		INSERT INTO users (login, password_hash, full_name, department_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, req.Login, passwordHash, req.FullName, depID).Scan(&userID)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	for _, role := range req.Roles {
		_, err := tx.Exec(`
			INSERT INTO user_roles (user_id, role) VALUES ($1, $2)
		`, userID, role)
		if err != nil {
			return nil, fmt.Errorf("failed to add role %s: %w", role, err)
		}
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
		UPDATE users SET login = $1, full_name = $2, is_active = $3, department_id = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
	`, req.Login, req.FullName, req.IsActive, depID, uid)

	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Обновляем роли: удаляем старые, добавляем новые
	_, err = tx.Exec(`DELETE FROM user_roles WHERE user_id = $1`, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to clear roles: %w", err)
	}

	for _, role := range req.Roles {
		_, err := tx.Exec(`
			INSERT INTO user_roles (user_id, role) VALUES ($1, $2)
		`, uid, role)
		if err != nil {
			return nil, fmt.Errorf("failed to add role %s: %w", role, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetByID(uid)
}

// GetUserRoles возвращает список ролей пользователя по его ID.
func (r *UserRepository) GetUserRoles(userID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT role FROM user_roles WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	roles := make([]string, 0)
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return roles, nil
}

// GetExecutors возвращает список всех активных пользователей с ролью 'executor' (исполнитель).
func (r *UserRepository) GetExecutors() ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT u.id, u.login, u.full_name, u.is_active, u.created_at, u.updated_at,
		       d.id, d.name
		FROM users u
		JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN departments d ON u.department_id = d.id
		WHERE ur.role = 'executor' AND u.is_active = true
		ORDER BY u.full_name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get executors: %w", err)
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
			&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
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

	// Batch-загрузка ролей одним запросом
	if err := r.batchLoadUserRoles(users, userIDs); err != nil {
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
		UPDATE users SET password_hash = $1, updated_at = CURRENT_TIMESTAMP
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

// batchLoadUserRoles загружает роли для всех пользователей одним SQL-запросом
// вместо N отдельных запросов (решение проблемы N+1).
func (r *UserRepository) batchLoadUserRoles(users []models.User, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}

	// Создаём индекс userID -> позиция в слайсе users
	userIndex := make(map[uuid.UUID]int, len(userIDs))
	for i, uid := range userIDs {
		userIndex[uid] = i
	}

	rows, err := r.db.Query(`
		SELECT user_id, role FROM user_roles WHERE user_id = ANY($1)
	`, pq.Array(userIDs))
	if err != nil {
		return fmt.Errorf("failed to batch load user roles: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID uuid.UUID
		var role string
		if err := rows.Scan(&userID, &role); err != nil {
			return err
		}
		if idx, ok := userIndex[userID]; ok {
			users[idx].Roles = append(users[idx].Roles, role)
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
