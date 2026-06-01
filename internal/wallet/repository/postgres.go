package repository

import (
	"database/sql"
	"fmt"

	"mangatyapi/internal/wallet/domain"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetBalance(userID string) (*domain.Wallet, error) {
	wallet := &domain.Wallet{}
	query := `
		SELECT id, user_id, balance, updated_at
		FROM wallets
		WHERE user_id = $1
	`
	err := r.db.QueryRow(query, userID).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Balance, &wallet.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("wallet no encontrada")
	}
	if err != nil {
		return nil, fmt.Errorf("error consultando wallet: %w", err)
	}
	
	return wallet, nil
}

func (r *PostgresRepository) GetWalletIDByUserID(userID string) (string, error) {
	var walletID string
	query := `SELECT id FROM wallets WHERE user_id = $1`
	err := r.db.QueryRow(query, userID).Scan(&walletID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("wallet no encontrada")
	}
	if err != nil {
		return "", err
	}
	return walletID, nil
}

func (r *PostgresRepository) UnlockChapter(userID, chapterID string, amount int) (*domain.Transaction, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("error iniciando transacción: %w", err)
	}
	defer tx.Rollback()

	// 1. Bloquear la wallet del usuario para evitar race conditions
	var walletID string
	var currentBalance int
	err = tx.QueryRow(`
		SELECT id, balance FROM wallets 
		WHERE user_id = $1 
		FOR UPDATE
	`, userID).Scan(&walletID, &currentBalance)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("wallet no encontrada para el usuario")
		}
		return nil, fmt.Errorf("error bloqueando wallet: %w", err)
	}

	// 2. Verificar saldo suficiente
	if currentBalance < amount {
		return nil, fmt.Errorf("saldo insuficiente. Tienes %d TyCoins, necesitas %d TyCoins", currentBalance, amount)
	}

	// 3. Verificar que el capítulo no esté ya desbloqueado
	var alreadyUnlocked bool
	err = tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM unlocked_chapters 
			WHERE user_id = $1 AND chapter_id = $2
		)
	`, userID, chapterID).Scan(&alreadyUnlocked)
	if err != nil {
		return nil, fmt.Errorf("error verificando desbloqueo: %w", err)
	}
	if alreadyUnlocked {
		return nil, fmt.Errorf("ya tienes este capítulo desbloqueado")
	}

	// 4. Verificar que el capítulo existe y obtener su precio real
	var actualPrice int
	var chapterTitle string
	err = tx.QueryRow(`
		SELECT price_tycoins, COALESCE(title, 'Sin título')
		FROM chapters 
		WHERE id = $1
	`, chapterID).Scan(&actualPrice, &chapterTitle)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("capítulo no encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("error verificando capítulo: %w", err)
	}

	// 5. Verificar que el precio coincida (protección contra manipulación)
	if actualPrice != amount {
		return nil, fmt.Errorf("el precio del capítulo ha cambiado. Actual: %d TyCoins", actualPrice)
	}

	// 6. Verificar que no sea gratuito
	if actualPrice == 0 {
		return nil, fmt.Errorf("este capítulo es gratuito, no requiere desbloqueo")
	}

	// 7. Descontar TyCoins de la wallet
	result, err := tx.Exec(`
		UPDATE wallets 
		SET balance = balance - $1, updated_at = CURRENT_TIMESTAMP 
		WHERE id = $2 AND balance >= $1
	`, amount, walletID)
	if err != nil {
		return nil, fmt.Errorf("error actualizando balance: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, fmt.Errorf("no se pudo actualizar el balance")
	}

	// 8. Registrar la transacción
	transaction := &domain.Transaction{
		WalletID:        walletID,
		ChapterID:       chapterID,
		Amount:          amount,
		TransactionType: "unlock",
	}

	err = tx.QueryRow(`
		INSERT INTO transactions (wallet_id, chapter_id, amount, transaction_type)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, transaction.WalletID, transaction.ChapterID, transaction.Amount, transaction.TransactionType).
		Scan(&transaction.ID, &transaction.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("error registrando transacción: %w", err)
	}

	// 9. Registrar capítulo desbloqueado
	_, err = tx.Exec(`
		INSERT INTO unlocked_chapters (user_id, chapter_id, transaction_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, chapter_id) DO NOTHING
	`, userID, chapterID, transaction.ID)
	if err != nil {
		return nil, fmt.Errorf("error registrando desbloqueo: %w", err)
	}

	// 10. Confirmar transacción
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error confirmando transacción: %w", err)
	}

	return transaction, nil
}


func (r *PostgresRepository) GetNewBalance(userID string) (int, error) {
	var balance int
	err := r.db.QueryRow(`SELECT balance FROM wallets WHERE user_id = $1`, userID).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("error consultando nuevo balance: %w", err)
	}
	return balance, nil
}

func (r *PostgresRepository) GetWalletByUserID(userID string) (*domain.Wallet, error) {
	wallet := &domain.Wallet{}
	query := `
		SELECT id, user_id, balance, updated_at
		FROM wallets
		WHERE user_id = $1
	`
	err := r.db.QueryRow(query, userID).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Balance, &wallet.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("wallet no encontrada para el usuario")
	}
	if err != nil {
		return nil, fmt.Errorf("error consultando wallet: %w", err)
	}
	return wallet, nil
}

func (r *PostgresRepository) GetChapterPrice(chapterID string) (int, error) {
	var price int
	err := r.db.QueryRow(`SELECT price_tycoins FROM chapters WHERE id = $1`, chapterID).Scan(&price)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("capítulo no encontrado")
	}
	if err != nil {
		return 0, fmt.Errorf("error consultando precio: %w", err)
	}
	return price, nil
}


func (r *PostgresRepository) IsChapterAlreadyUnlocked(userID, chapterID string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM unlocked_chapters 
			WHERE user_id = $1 AND chapter_id = $2
		)
	`
	err := r.db.QueryRow(query, userID, chapterID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error verificando desbloqueo previo: %w", err)
	}
	return exists, nil
}

