package usecase

import (
	"errors"
	"fmt"

	"mangatyapi/internal/auth/domain"
	"mangatyapi/pkg/security"
)

type Repository interface {
	CreateUser(user *domain.User) error
	GetUserByEmail(email string) (*domain.User, error)
	GetUserByID(userID string) (*domain.User, error)
	CreateWallet(userID string) error
	GetWalletByUserID(userID string) (string, error)
	AddUserRole(userID, roleName, grantedByID string) error
	RemoveUserRole(userID, roleName string) error
	HasRole(userID, roleName string) (bool, error)
	GetUserRoles(userID string) ([]string, error)
}

type UseCase struct {
	repo Repository
}

func NewUseCase(repo Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (uc *UseCase) Register(req domain.RegisterRequest) (*domain.AuthResponse, error) {
	// Hashear contraseña
	hashedPassword, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("error procesando contraseña")
	}

	// Crear usuario (rol lector se asigna automáticamente)
	user := &domain.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
	}

	if err := uc.repo.CreateUser(user); err != nil {
		return nil, err
	}

	// Crear wallet automáticamente
	if err := uc.repo.CreateWallet(user.ID); err != nil {
		return nil, errors.New("error creando billetera virtual")
	}

	return &domain.AuthResponse{
		Status:  "success",
		Message: "Usuario registrado exitosamente. Ya puedes leer y publicar tus propios cómics.",
		UserID:  user.ID,
	}, nil
}

func (uc *UseCase) Login(req domain.LoginRequest) (*domain.AuthResponse, error) {
	user, err := uc.repo.GetUserByEmail(req.Email)
	if err != nil {
		return nil, errors.New("credenciales inválidas")
	}

	if !security.CheckPassword(req.Password, user.PasswordHash) {
		return nil, errors.New("credenciales inválidas")
	}

	// Generar JWT con todos los roles
	token, err := security.GenerateJWT(user.ID, user.Email, user.Roles)
	if err != nil {
		return nil, errors.New("error generando token")
	}

	return &domain.AuthResponse{
		Status:    "success",
		Token:     token,
		ExpiresIn: "24h",
	}, nil
}

// Nuevo: Admin asigna rol a otro usuario
func (uc *UseCase) AssignRole(adminID, targetUserID, roleName string) (*domain.AuthResponse, error) {
	// Verificar que el admin tenga rol de administrador
	isAdmin, err := uc.repo.HasRole(adminID, "administrador")
	if err != nil || !isAdmin {
		return nil, errors.New("no tienes permisos de administrador")
	}

	// Verificar que el rol existe
	if roleName != "autor" && roleName != "administrador" {
		return nil, errors.New("rol no válido")
	}

	if err := uc.repo.AddUserRole(targetUserID, roleName, adminID); err != nil {
		return nil, err
	}

	return &domain.AuthResponse{
		Status:  "success",
		Message: fmt.Sprintf("Rol '%s' asignado exitosamente", roleName),
	}, nil
}

// Nuevo: Obtener perfil con roles
func (uc *UseCase) GetProfile(userID string) (*domain.UserProfile, error) {
	user, err := uc.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	return &domain.UserProfile{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Roles:     user.Roles,
		CreatedAt: user.CreatedAt,
	}, nil
}
