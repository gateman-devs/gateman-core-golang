package wallet

import (
	"errors"
	"fmt"

	"gateman.io/application/repository"
	"gateman.io/entities"
	"gateman.io/infrastructure/logger"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	LIVENESS_PRICE  = 5000  // 50_00 kobo
	COMPARISON_PRICE = 7000 // 70_00 kobo
)

type WalletService struct{}

func NewWalletService() *WalletService {
	return &WalletService{}
}

// CheckAndDeductBalance checks if workspace has sufficient balance and deducts the amount atomically
func (ws *WalletService) CheckAndDeductBalance(workspaceID string, amount int64, operation string, requestID string) (*entities.BillingLog, error) {
	workspaceRepo := repository.WorkspaceRepository()
	billingRepo := repository.BillingLogRepository()

	// Use MongoDB atomic operation to check and deduct balance
	filter := bson.M{
		"_id": workspaceID,
		"balance": bson.M{"$gte": amount},
	}

	update := bson.M{
		"$inc": bson.M{
			"balance": -amount,
		},
	}

	// Update workspace balance
	success, err := workspaceRepo.UpdateWithOperator(filter, update)
	if err != nil {
		logger.Error("Failed to deduct balance", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "workspaceID",
			Data: workspaceID,
		})
		return nil, fmt.Errorf("failed to deduct balance: %w", err)
	}

	if !success {
		return nil, errors.New("insufficient balance")
	}

	// Create billing log
	billingLog := &entities.BillingLog{
		WorkspaceID: workspaceID,
		RequestID:   requestID,
		Operation:   operation,
		Amount:      amount,
		Status:      "success",
	}

	createdBillingLog, err := billingRepo.CreateOne(nil, *billingLog)
	if err != nil {
		logger.Error("Failed to create billing log", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		// If billing log creation fails, we should revert the balance deduction
		revertFilter := bson.M{"_id": workspaceID}
		revertUpdate := bson.M{
			"$inc": bson.M{
				"balance": amount,
			},
		}
		workspaceRepo.UpdateWithOperator(revertFilter, revertUpdate)
		return nil, fmt.Errorf("failed to create billing log: %w", err)
	}

	return createdBillingLog, nil
}

// LockFunds locks funds for a pending operation
func (ws *WalletService) LockFunds(workspaceID string, amount int64) error {
	workspaceRepo := repository.WorkspaceRepository()

	filter := bson.M{
		"_id": workspaceID,
		"balance": bson.M{"$gte": amount},
	}

	update := bson.M{
		"$inc": bson.M{
			"balance":      -amount,
			"lockedBalance": amount,
		},
	}

	success, err := workspaceRepo.UpdateWithOperator(filter, update)
	if err != nil {
		return fmt.Errorf("failed to lock funds: %w", err)
	}

	if !success {
		return errors.New("insufficient balance to lock funds")
	}

	return nil
}

// UnlockFunds unlocks previously locked funds (on failure)
func (ws *WalletService) UnlockFunds(workspaceID string, amount int64) error {
	workspaceRepo := repository.WorkspaceRepository()

	filter := bson.M{
		"_id": workspaceID,
		"lockedBalance": bson.M{"$gte": amount},
	}

	update := bson.M{
		"$inc": bson.M{
			"balance":       amount,
			"lockedBalance": -amount,
		},
	}

	success, err := workspaceRepo.UpdateWithOperator(filter, update)
	if err != nil {
		return fmt.Errorf("failed to unlock funds: %w", err)
	}

	if !success {
		return errors.New("insufficient locked balance to unlock")
	}

	return nil
}

// ConfirmLockedFunds converts locked funds to permanent deduction (on success)
func (ws *WalletService) ConfirmLockedFunds(workspaceID string, amount int64) error {
	workspaceRepo := repository.WorkspaceRepository()

	filter := bson.M{
		"_id": workspaceID,
		"lockedBalance": bson.M{"$gte": amount},
	}

	update := bson.M{
		"$inc": bson.M{
			"lockedBalance": -amount,
		},
	}

	success, err := workspaceRepo.UpdateWithOperator(filter, update)
	if err != nil {
		return fmt.Errorf("failed to confirm locked funds: %w", err)
	}

	if !success {
		return errors.New("insufficient locked balance to confirm")
	}

	return nil
}

// GetBalance returns the current balance of a workspace
func (ws *WalletService) GetBalance(workspaceID string) (int64, error) {
	workspaceRepo := repository.WorkspaceRepository()
	workspace, err := workspaceRepo.FindByID(workspaceID)
	if err != nil {
		return 0, err
	}
	if workspace == nil {
		return 0, errors.New("workspace not found")
	}
	return workspace.Balance, nil
}
