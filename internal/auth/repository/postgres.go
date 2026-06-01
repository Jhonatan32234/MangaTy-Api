package repository

import (
	"database/sql"
	"fmt"

	"mangatyapi/internal/auth/domain"

	"github.com/lib/pq"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateUser(user *domain.User) error {
	query := `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	err := r.db.QueryRow(query, user.Username, user.Email, user.PasswordHash).
		Scan(&user.ID, &user.CreatedAt)

	if err != nil {
		return fmt.Errorf("error creando usuario: %w", err)
	}

	// Asignar roles de lector y autor por defecto para que el usuario pueda publicar desde el inicio
	_, err = r.db.Exec(`
		INSERT INTO user_roles (user_id, role_id)
		SELECT $1, id FROM roles WHERE name IN ('lector', 'autor')
	`, user.ID)

	if err != nil {
		return fmt.Errorf("error asignando rol de lector: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetUserByEmail(email string) (*domain.User, error) {
	user := &domain.User{}
	query := `
		SELECT u.id, u.username, u.email, u.password_hash, 
			   COALESCE(array_agg(r.name) FILTER (WHERE r.name IS NOT NULL), '{}') as roles,
			   u.created_at
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles r ON ur.role_id = r.id
		WHERE u.email = $1
		GROUP BY u.id
	`

	var roles []string
	err := r.db.QueryRow(query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		pq.Array(&roles), &user.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("usuario no encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("error buscando usuario: %w", err)
	}

	user.Roles = roles
	return user, nil
}

func (r *PostgresRepository) GetUserByID(userID string) (*domain.User, error) {
	user := &domain.User{}
	query := `
		SELECT u.id, u.username, u.email, u.password_hash,
			   COALESCE(array_agg(r.name) FILTER (WHERE r.name IS NOT NULL), '{}') as roles,
			   u.created_at
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles r ON ur.role_id = r.id
		WHERE u.id = $1
		GROUP BY u.id
	`

	var roles []string
	err := r.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		pq.Array(&roles), &user.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("usuario no encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("error buscando usuario: %w", err)
	}

	user.Roles = roles
	return user, nil
}

func (r *PostgresRepository) CreateWallet(userID string) error {
	query := `INSERT INTO wallets (user_id, balance) VALUES ($1, 0)`
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("error creando wallet: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetWalletByUserID(userID string) (string, error) {
	var walletID string
	query := `SELECT id FROM wallets WHERE user_id = $1`
	err := r.db.QueryRow(query, userID).Scan(&walletID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("wallet no encontrada")
	}
	if err != nil {
		return "", fmt.Errorf("error buscando wallet: %w", err)
	}
	return walletID, nil
}

// Nuevo: Agregar rol a usuario
func (r *PostgresRepository) AddUserRole(userID, roleName, grantedByID string) error {
	_, err := r.db.Exec(`
		INSERT INTO user_roles (user_id, role_id, granted_by)
		SELECT $1, r.id, $3
		FROM roles r
		WHERE r.name = $2
		ON CONFLICT (user_id, role_id) DO NOTHING
	`, userID, roleName, grantedByID)

	if err != nil {
		return fmt.Errorf("error agregando rol: %w", err)
	}
	return nil
}

// Nuevo: Remover rol de usuario
func (r *PostgresRepository) RemoveUserRole(userID, roleName string) error {
	// No permitir remover el rol de lector
	if roleName == "lector" {
		return fmt.Errorf("no se puede remover el rol básico de lector")
	}

	result, err := r.db.Exec(`
		DELETE FROM user_roles
		WHERE user_id = $1 
		AND role_id = (SELECT id FROM roles WHERE name = $2)
		AND role_id != (SELECT id FROM roles WHERE name = 'lector')
	`, userID, roleName)

	if err != nil {
		return fmt.Errorf("error removiendo rol: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("el usuario no tiene el rol especificado")
	}

	return nil
}

// Nuevo: Verificar si un usuario tiene un rol específico
func (r *PostgresRepository) HasRole(userID, roleName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_roles ur
			JOIN roles r ON ur.role_id = r.id
			WHERE ur.user_id = $1 AND r.name = $2
		)
	`
	err := r.db.QueryRow(query, userID, roleName).Scan(&exists)
	return exists, err
}

// Nuevo: Obtener todos los roles de un usuario
func (r *PostgresRepository) GetUserRoles(userID string) ([]string, error) {
	var roles []string
	query := `
		SELECT array_agg(r.name)
		FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1
	`
	err := r.db.QueryRow(query, userID).Scan(pq.Array(&roles))
	if err != nil {
		return nil, err
	}
	return roles, nil
}
