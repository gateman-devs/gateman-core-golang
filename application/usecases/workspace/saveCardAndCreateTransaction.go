package workspace_usecases

import (
	"context"

	"gateman.io/application/repository"
	"gateman.io/application/utils"
	"gateman.io/entities"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/logger"
	paystack_local_payment_processor "gateman.io/infrastructure/payments/paystack"
)

func SaveCardAndCreateTransaction(ctx *any, trxDescription string, transaction paystack_local_payment_processor.TransactionData) {
	transactionRepo := repository.TransactionRepo()
	cardAuthCode := transaction.Authorization.AuthorizationCode
	transaction.Authorization.AuthorizationCode = ""
	transactionRepo.CreateOne(context.TODO(), entities.Transaction{
		AppID:       &transaction.Metadata.AppID,
		RefID:       transaction.Reference,
		WorkspaceID: transaction.Metadata.WorkspaceID,
		Amount:      uint32(transaction.Amount),
		PlanID:      &transaction.Metadata.PlanID,
		Description: &trxDescription,
		Metadata:    transaction,
	})

	workspaceRepo := repository.WorkspaceRepository()
	workspace, _ := workspaceRepo.FindByID(transaction.Metadata.WorkspaceID)
	exists := false
	var newCard *entities.CardInfo
	for _, card := range workspace.PaymentDetails {
		if card.Signature == *transaction.Authorization.Signature {
			exists = true
			newCard = &card
			break
		}
	}
	if !exists {
		encryptedAuthCode, err := cryptography.EncryptData([]byte(cardAuthCode), nil)
		if err != nil {
			logger.Error("an error occured while encrypting card auth code", logger.LoggerOptions{
				Key:  "payload",
				Data: transaction,
			})
			return
		}
		newCard = &entities.CardInfo{
			ID:                utils.GenerateUULDString(),
			AuthorizationCode: *encryptedAuthCode,
			Bin:               transaction.Authorization.Bin,
			Last4:             transaction.Authorization.Last4,
			ExpMonth:          transaction.Authorization.ExpMonth,
			ExpYear:           transaction.Authorization.ExpYear,
			Channel:           transaction.Authorization.Channel,
			CardType:          transaction.Authorization.CardType,
			Bank:              *transaction.Authorization.Bank,
			CountryCode:       transaction.Authorization.CountryCode,
			Brand:             transaction.Authorization.Brand,
			Reusable:          transaction.Authorization.Reusable,
			AccountName:       transaction.Authorization.AccountName,
			Signature:         *transaction.Authorization.Signature,
		}
		workspace.PaymentDetails = append(workspace.PaymentDetails, *newCard)
		workspaceRepo.UpdatePartialByID(workspace.ID, map[string]any{
			"paymentDetails": workspace.PaymentDetails,
		})
	}
	if transaction.Metadata.AppID != "" {
		appRepo := repository.ApplicationRepo()
		appRepo.UpdatePartialByID(transaction.Metadata.AppID, map[string]any{
			"paymentCard": newCard.ID,
		})
	}
}
