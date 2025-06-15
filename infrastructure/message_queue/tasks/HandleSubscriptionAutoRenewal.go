package queue_tasks

import (
	"context"
	"encoding/json"
	"errors"

	"gateman.io/application/repository"
	"gateman.io/entities"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/logger"
	mq_types "gateman.io/infrastructure/message_queue/types"
	"gateman.io/infrastructure/payments"
	"github.com/hibiken/asynq"
)

var HandleSubscriptionAutoRenewal mq_types.Queues = "subscription_auto_renewal"

type RenewSubscriptionPayload struct {
	AppID string
	mq_types.BasePayload
}

func HandleSubsciptionAutoRenewalTask(ctx context.Context, t *asynq.Task) error {
	var payload RenewSubscriptionPayload
	err := json.Unmarshal(t.Payload(), &payload)
	if err != nil {
		logger.Error("an error occured while unmarshalling sub auto renewal queue payload", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return err
	}
	appRepo := repository.ApplicationRepo()
	app, err := appRepo.FindByID(payload.AppID)
	if err != nil {
		logger.Error("an error occured while fetch app sub auto renewal queue payload", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{Key: "payload", Data: payload})
		return err
	}
	if app == nil {
		logger.Error("app does not exist", logger.LoggerOptions{
			Key:  "appID",
			Data: payload.AppID,
		})
		return asynq.SkipRetry
	}

	if app.PaymentCard == nil {
		logger.Error("payment card not set on app", logger.LoggerOptions{
			Key:  "appID",
			Data: payload.AppID,
		})
		return errors.New("payment card not set on app")
	}

	workspaceRepo := repository.WorkspaceRepository()
	workspace, _ := workspaceRepo.FindByID(app.WorkspaceID)
	if workspace == nil {
		logger.Error("invalid workspace id", logger.LoggerOptions{
			Key:  "appID",
			Data: payload.AppID,
		})
		return asynq.SkipRetry
	}
	var card *entities.CardInfo
	for _, savedCard := range workspace.PaymentDetails {
		if savedCard.ID == *app.PaymentCard {
			card = &savedCard
			break
		}
	}
	if card == nil {
		logger.Error("invalid payment card selected for subcription payment", logger.LoggerOptions{
			Key: "app", Data: app.ID,
		}, logger.LoggerOptions{Key: "paymentID", Data: app.PaymentCard})
		return errors.New("invalid payment card selected for subcription payment")
	}
	subscriptionRepo := repository.SubscriptionPlanRepo()
	activeSubRepo := repository.ActiveSubscriptionRepo()
	activeSub, _ := activeSubRepo.FindOneByFilter(map[string]any{
		"appID": app.ID,
	})
	if activeSub == nil {
		logger.Error("active subscription not found", logger.LoggerOptions{
			Key: "app", Data: app.ID,
		})

		return errors.New("active subscription not found")
	}

	if !activeSub.AutoRenew {
		logger.Error("autorenewal has been turned off", logger.LoggerOptions{
			Key: "active sub id", Data: activeSub.ID,
		})
		return asynq.SkipRetry
	}

	sub, _ := subscriptionRepo.FindByID(activeSub.SubscriptionID)

	if sub == nil {
		logger.Error("subscription not found", logger.LoggerOptions{
			Key: "app", Data: app.ID,
		}, logger.LoggerOptions{Key: "subscription id", Data: activeSub.SubscriptionID})
		return errors.New("subscription not found")
	}
	var amount uint32
	if activeSub.Interval == entities.Annually {
		amount = sub.AnnualPrice
	} else {
		amount = sub.MonthlyPrice
	}

	authCode, _ := cryptography.DecryptData(card.AuthorizationCode, nil)
	payments.PaymentProcessor.ChargeCard(string(authCode), workspace.Email, amount, map[string]any{
		"workspaceID": workspace.ID,
		"appID":       app.ID,
		"planID":      activeSub.SubscriptionID,
		"frequency":   activeSub.Interval,
		"autoRenew":   true,
	})
	return nil
}
