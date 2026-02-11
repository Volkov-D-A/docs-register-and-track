package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"docflow/internal/database"
	"docflow/internal/models"
)

type UserRepository struct {
	db *database.DB
}

func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByLogin(login string) (*models.User, error) {
	user := &models.User{}

	err := r.db.QueryRow(`
		SELECT id, login, password_hash, full_name, is_active, created_at, updated_at
		FROM users WHERE login = $1
	`, login).Scan(
		&user.ID, &user.Login, &user.PasswordHash, &user.FullName,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by login: %w", err)
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

	err := r.db.QueryRow(`
		SELECT id, login, password_hash, full_name, is_active, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(
		&user.ID, &user.Login, &user.PasswordHash, &user.FullName,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
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
		SELECT id, login, password_hash, full_name, is_active, created_at, updated_at
		FROM users ORDER BY full_name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID, &user.Login, &user.PasswordHash, &user.FullName,
			&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		); err != nil {
			return nil, err
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
	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	var userID uuid.UUID
	err = r.db.QueryRow(`
		INSERT INTO users (login, password_hash, full_name)
		VALUES ($1, $2, $3)
		RETURNING id
	`, req.Login, passwordHash, req.FullName).Scan(&userID)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	for _, role := range req.Roles {
		_, err := r.db.Exec(`
			INSERT INTO user_roles (user_id, role) VALUES ($1, $2)
		`, userID, role)
		if err != nil {
			return nil, fmt.Errorf("failed to add role %s: %w", role, err)
		}
	}

	return r.GetByID(userID)
}

func (r *UserRepository) Update(req models.UpdateUserRequest) (*models.User, error) {
	uid, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	_, err = r.db.Exec(`
		UPDATE users SET login = $1, full_name = $2, is_active = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
	`, req.Login, req.FullName, req.IsActive, uid)

	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Обновляем роли: удаляем старые, добавляем новые
	_, err = r.db.Exec(`DELETE FROM user_roles WHERE user_id = $1`, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to clear roles: %w", err)
	}

	for _, role := range req.Roles {
		_, err := r.db.Exec(`
			INSERT INTO user_roles (user_id, role) VALUES ($1, $2)
		`, uid, role)
		if err != nil {
			return nil, fmt.Errorf("failed to add role %s: %w", role, err)
		}
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
		SELECT DISTINCT u.id, u.login, u.password_hash, u.full_name, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN user_roles ur ON u.id = ur.user_id
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
		if err := rows.Scan(
			&user.ID, &user.Login, &user.PasswordHash, &user.FullName,
			&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		); err != nil {
			return nil, err
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
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	return r.UpdatePassword(userID, hash)
}

func VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}
