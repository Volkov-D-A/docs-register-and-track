package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"docflow/internal/database"
	"docflow/internal/models"
	"docflow/internal/security"
)

type UserRepository struct {
	db *database.DB
}

func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByLogin(login string) (*models.User, error) {
	user := &models.User{}

	var departmentID sql.NullString
	var departmentName sql.NullString

	err := r.db.QueryRow(`
		SELECT u.id, u.login, u.password_hash, u.full_name, u.is_active, u.created_at, u.updated_at,
		       d.id, d.name
		FROM users u
		LEFT JOIN departments d ON u.department_id = d.id
		WHERE u.login = $1
	`, login).Scan(
		&user.ID, &user.Login, &user.PasswordHash, &user.FullName,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		&departmentID, &departmentName,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by login: %w", err)
	}

	if departmentID.Valid {
		uid, _ := uuid.Parse(departmentID.String)
		user.DepartmentID = &uid
		user.Department = &models.Department{
			ID:    uid,
			IDStr: uid.String(),
			Name:  departmentName.String,
		}
		if noms, err := r.getDepartmentNomenclatureIDs(uid); err == nil {
			user.Department.NomenclatureIDs = noms
		}
	}

	user.FillIDStr()

	roles, err := r.GetUserRoles(user.ID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	return user, nil
}

func (r *UserRepository) GetByID(id uuid.UUID) (*models.User, error) {
	user := &models.User{}

	var departmentID sql.NullString
	var departmentName sql.NullString

	err := r.db.QueryRow(`
		SELECT u.id, u.login, u.password_hash, u.full_name, u.is_active, u.created_at, u.updated_at,
		       d.id, d.name
		FROM users u
		LEFT JOIN departments d ON u.department_id = d.id
		WHERE u.id = $1
	`, id).Scan(
		&user.ID, &user.Login, &user.PasswordHash, &user.FullName,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		&departmentID, &departmentName,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	if departmentID.Valid {
		uid, _ := uuid.Parse(departmentID.String)
		user.DepartmentID = &uid
		user.Department = &models.Department{
			ID:    uid,
			IDStr: uid.String(),
			Name:  departmentName.String,
		}
		if noms, err := r.getDepartmentNomenclatureIDs(uid); err == nil {
			user.Department.NomenclatureIDs = noms
		}
	}

	user.FillIDStr()

	roles, err := r.GetUserRoles(user.ID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	return user, nil
}

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

	var users []models.User
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
				ID:    uid,
				IDStr: uid.String(),
				Name:  departmentName.String,
			}
		}
		if user.Department != nil {
			if noms, err := r.getDepartmentNomenclatureIDs(user.Department.ID); err == nil {
				user.Department.NomenclatureIDs = noms
			}
		}

		user.FillIDStr()

		roles, err := r.GetUserRoles(user.ID)
		if err != nil {
			return nil, err
		}
		user.Roles = roles
		users = append(users, user)
	}

	return users, nil
}

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

func (r *UserRepository) GetUserRoles(userID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT role FROM user_roles WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	return roles, nil
}

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

	var users []models.User
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
				ID:    uid,
				IDStr: uid.String(),
				Name:  departmentName.String,
			}
		}
		if user.Department != nil {
			if noms, err := r.getDepartmentNomenclatureIDs(user.Department.ID); err == nil {
				user.Department.NomenclatureIDs = noms
			}
		}

		user.FillIDStr()
		roles, err := r.GetUserRoles(user.ID)
		if err != nil {
			return nil, err
		}
		user.Roles = roles
		users = append(users, user)
	}

	return users, nil
}

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

	var ids []string
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id.String())
	}
	return ids, nil
}

// CountUsers — получить общее количество пользователей в базе данных
func (r *UserRepository) CountUsers() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}
