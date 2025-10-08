package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	"gateman.io/application/repository"
	workspace_usecases "gateman.io/application/usecases/workspace"
	"gateman.io/application/utils"
	"gateman.io/entities"
	"gateman.io/infrastructure/logger"
	messagequeue "gateman.io/infrastructure/message_queue"
	queue_tasks "gateman.io/infrastructure/message_queue/tasks"
	mq_types "gateman.io/infrastructure/message_queue/types"
	"gateman.io/infrastructure/payments"
	paystack_local_payment_processor "gateman.io/infrastructure/payments/paystack"
	server_response "gateman.io/infrastructure/serverResponse"
)

func ProcessPaystackWebhook(ctx *interfaces.ApplicationContext[[]byte]) {
	hash := utils.CreateHMACSHA512Hash(*ctx.Body, os.Getenv("PAYSTACK_ACCESS_TOKEN"))
	if hash != *ctx.GetHeader("X-Paystack-Signature") {
		logger.Error("invalid payload and hash from webhook", logger.LoggerOptions{
			Key:  "payload",
			Data: ctx.Body,
		})
		apperrors.ClientError(ctx.Ctx, "webhook failed", nil, nil, ctx.DeviceID)
		return
	}
	var body dto.PaystackWebhookDTO
	err := json.Unmarshal(*ctx.Body, &body)
	if err != nil {
		logger.Error("an error occured while serializing paystack webhook to a struct", logger.LoggerOptions{
			Key: "err", Data: err,
		})
		apperrors.ClientError(ctx.Ctx, "an error occured while serializing paystack webhook", nil, nil, ctx.DeviceID)
		return
	}
	if body.Event == "charge.success" {
		verifiedDataAny, err := payments.PaymentProcessor.VerifyTransaction(body.Data.Reference)
		if err != nil {
			logger.Error("an error occured while verifying transaction", logger.LoggerOptions{
				Key:  "payload",
				Data: body,
			})
			apperrors.ClientError(ctx.Ctx, "an error occured while verifying transaction", nil, nil, ctx.DeviceID)
			return
		}
		verifiedData := verifiedDataAny.(paystack_local_payment_processor.TransactionData)
		if verifiedData.Status != "success" {
			logger.Error("transaction failed", logger.LoggerOptions{
				Key:  "payload",
				Data: body,
			})
			apperrors.ClientError(ctx.Ctx, "transaction failed", nil, nil, ctx.DeviceID)
			return
		}
		transactionRepo := repository.TransactionRepo()
		trxExists, _ := transactionRepo.CountDocs(map[string]interface{}{"refID": verifiedData.Reference})
		if trxExists != 0 {
			logger.Error("webhook rejected due to duplicate transaction", logger.LoggerOptions{
				Key:  "body",
				Data: body,
			})
			server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "webhook already processed", nil, nil, nil, &ctx.DeviceID)
			return
		}
		if verifiedData.Metadata.Reverse == "true" {
			workspace_usecases.SaveCardAndCreateTransaction(&ctx.Ctx, "Card verification attempt", verifiedData)
			payments.PaymentProcessor.ReverseTransaction(verifiedData.Reference, "Card verification charge reversal")
			server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "processed successfully", nil, nil, nil, &ctx.DeviceID)
			return
		}
		subscriptionRepo := repository.SubscriptionPlanRepo()
		subscription, _ := subscriptionRepo.FindByID(verifiedData.Metadata.PlanID)
		activeSubscriptionRepo := repository.ActiveSubscriptionRepo()
		activeSub, _ := activeSubscriptionRepo.FindOneByFilter(map[string]interface{}{
			"appID": verifiedData.Metadata.AppID,
		})
		now := time.Now()
		var expireAfter int
		if verifiedData.Metadata.Frequency == "monthly" {
			expireAfter = 30
		} else {
			expireAfter = 365
		}
		expiresOn := now.AddDate(0, 0, expireAfter)
		if activeSub == nil {
			subscriptionPlanRepo := repository.SubscriptionPlanRepo()
			subscription, err := subscriptionPlanRepo.FindByID(verifiedData.Metadata.PlanID)
			if err != nil {
				logger.Error("an error occured while fetching new subscription id after payment", logger.LoggerOptions{
					Key:  "error",
					Data: err,
				})
				apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
				return
			}
			if subscription == nil {
				apperrors.NotFoundError(ctx.Ctx, "Subscription not found", &ctx.DeviceID)
				return
			}
			activeSubscriptionRepo.CreateOne(context.TODO(), entities.ActiveSubscription{
				AppID:          verifiedData.Metadata.AppID,
				SubscriptionID: subscription.ID,
				ActiveSubID:    subscription.ID,
				Name:           subscription.Name,
				ActiveSubName:  subscription.Name,
				Active:         true,
				AutoRenew:      verifiedData.Metadata.AutoRenew == "true",
				WorkspaceID:    verifiedData.Metadata.WorkspaceID,
				RenewedOn:      &now,
				ExpiresOn:      &expiresOn,
				Interval:       entities.SubscriptionFrequency(verifiedData.Metadata.Frequency),
			})
		} else {
			activeSubscriptionRepo.UpdatePartialByID(activeSub.ID, map[string]any{
				"interval":       entities.SubscriptionFrequency(verifiedData.Metadata.Frequency),
				"expiresOn":      &expiresOn,
				"renewedOn":      &now,
				"autoRenew":      verifiedData.Metadata.AutoRenew == "true",
				"active":         true,
				"activeSubName":  subscription.Name,
				"name":           subscription.Name,
				"subscriptionID": subscription.ID,
				"activeSubID":    subscription.ID,
			})
		}
		workspace_usecases.SaveCardAndCreateTransaction(&ctx.Ctx, fmt.Sprintf("Gateman %s - %s", subscription.Name, verifiedData.Metadata.Frequency), verifiedData)
		if activeSub == nil || activeSub.AutoRenew {
			renewSubPayload, err := json.Marshal(queue_tasks.RenewSubscriptionPayload{
				AppID: verifiedData.Metadata.AppID,
				BasePayload: mq_types.BasePayload{
					RetryInterval: time.Hour * 24,
				},
			})
			if err != nil {
				logger.Error("error marshalling payload for sub auto renewal queue")
				apperrors.ErrorProcessingPayload(ctx, nil)
				return
			}
			messagequeue.TaskQueue.Enqueue(mq_types.QueueTask{
				Payload:   renewSubPayload,
				Name:      queue_tasks.HandleSubscriptionAutoRenewal,
				Priority:  mq_types.High,
				MaxRetry:  30,
				ProcessIn: (time.Minute * time.Duration(24*expireAfter)),
			})
		}
	} else {
		logger.Error("an unsupported event was sent by paystack", logger.LoggerOptions{
			Key:  "payload",
			Data: body,
		})
		apperrors.ClientError(ctx.Ctx, "an unknown event was emitted", nil, nil, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "processed successfully", nil, nil, nil, &ctx.DeviceID)
}
