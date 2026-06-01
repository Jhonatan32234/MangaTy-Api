package usecase

import (
	"fmt"

	"mangatyapi/internal/wallet/domain"
)

type Repository interface {
	GetBalance(userID string) (*domain.Wallet, error)
	GetWalletByUserID(userID string) (*domain.Wallet, error)
	GetChapterPrice(chapterID string) (int, error)
	IsChapterAlreadyUnlocked(userID, chapterID string) (bool, error)
	UnlockChapter(userID, chapterID string, amount int) (*domain.Transaction, error)
	GetNewBalance(userID string) (int, error)

}

type UseCase struct {
	repo Repository
}

func NewUseCase(repo Repository) *UseCase {
	return &UseCase{repo: repo}
}


func (uc *UseCase) GetBalance(userID string) (*domain.BalanceResponse, error) {
	wallet, err := uc.repo.GetBalance(userID)
	if err != nil {
		return nil, err
	}

	return &domain.BalanceResponse{
		WalletID: wallet.ID,
		Balance:  wallet.Balance,
		Currency: "TyCoins",
	}, nil
}

func (uc *UseCase) UnlockChapter(userID, chapterID string) (*domain.UnlockChapterResponse, error) {
	// 1. Verificar que el usuario tenga wallet
	wallet, err := uc.repo.GetWalletByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("no tienes una billetera virtual")
	}

	// 2. Verificar que el capítulo existe
	price, err := uc.repo.GetChapterPrice(chapterID)
	if err != nil {
		return nil, err
	}

	// 3. Verificar que no sea gratuito
	if price == 0 {
		return nil, fmt.Errorf("este capítulo es gratuito, no necesitas desbloquearlo")
	}

	// 4. Verificar que no lo haya comprado antes
	alreadyUnlocked, err := uc.repo.IsChapterAlreadyUnlocked(userID, chapterID)
	if err != nil {
		return nil, err
	}
	if alreadyUnlocked {
		return nil, fmt.Errorf("ya tienes este capítulo desbloqueado")
	}

	// 5. Verificar saldo suficiente
	if wallet.Balance < price {
		return nil, fmt.Errorf("saldo insuficiente. Tienes %d TyCoins, necesitas %d TyCoins", wallet.Balance, price)
	}

	// 6. Procesar la transacción
	transaction, err := uc.repo.UnlockChapter(userID, chapterID, price)
	if err != nil {
		return nil, err
	}

	// 7. Obtener nuevo balance
	newBalance, err := uc.repo.GetNewBalance(userID)
	if err != nil {
		return nil, err
	}

	return &domain.UnlockChapterResponse{
		Status:        "approved",
		TransactionID: transaction.ID,
		NewBalance:    newBalance,
		Message:       fmt.Sprintf("¡Capítulo desbloqueado! Has gastado %d TyCoins. Tu nuevo saldo es %d TyCoins.", price, newBalance),
	}, nil
}