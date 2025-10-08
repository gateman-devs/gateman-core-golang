package controller

import (
	"fmt"
	"math"
	"net/http"
	"time"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	"gateman.io/application/repository"
	"gateman.io/application/utils"
	"gateman.io/entities"
	fileupload "gateman.io/infrastructure/file_upload"
	"gateman.io/infrastructure/file_upload/types"
	"gateman.io/infrastructure/logger"
	"gateman.io/infrastructure/payments"
	payment_types "gateman.io/infrastructure/payments/types"
	server_response "gateman.io/infrastructure/serverResponse"
	"gateman.io/infrastructure/validator"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GeneratedSignedURL(ctx *interfaces.ApplicationContext[dto.GeneratedSignedURLDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	if ctx.Body.AccountImage {
		ctx.Body.FilePath = fmt.Sprintf("%s/%s", ctx.GetStringContextData("UserID"), "accountimage")
	}
	var url *string
	var err error
	if ctx.Body.Permission.Read {
		url, err = fileupload.FileUploader.GeneratedSignedURL(ctx.Body.FilePath, types.SignedURLPermission{
			Read: true,
		}, time.Minute*1)
	} else if ctx.Body.Permission.Write {
		url, err = fileupload.FileUploader.GeneratedSignedURL(ctx.Body.FilePath, types.SignedURLPermission{
			Write: true,
		}, time.Minute*1)
	} else if ctx.Body.Permission.Delete {
	} else {
		apperrors.ClientError(ctx.Ctx, "invalid request", nil, nil, ctx.DeviceID)
		return
	}
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "account created", map[string]any{
		"url":      url,
		"filePath": ctx.Body.FilePath,
	}, nil, nil, &ctx.DeviceID)
}

func GetSubscriptionPlans(ctx *interfaces.ApplicationContext[any]) {
	subscriptionRepo := repository.SubscriptionPlanRepo()
	plans, err := subscriptionRepo.FindMany(map[string]interface{}{})
	if err != nil {
		logger.Error("an error occured while fetch subscription plans", logger.LoggerOptions{
			Key:  "err",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "plans fetched", *plans, nil, nil, &ctx.DeviceID)
}

func GenerateLinkToAddCard(ctx *interfaces.ApplicationContext[dto.GenerateAddCardLinkDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	var appEmail *string
	if ctx.Body.AppID != nil {
		appRepo := repository.ApplicationRepo()
		app, err := appRepo.FindByID(*ctx.Body.AppID, options.FindOne().SetProjection(map[string]any{
			"email": 1,
		}))
		if err != nil {
			logger.Error("an error occured while fetching app to generate payment link", logger.LoggerOptions{
				Key:  "err",
				Data: err,
			})
			apperrors.UnknownError(ctx.Ctx, nil, nil, ctx.DeviceID)
			return
		}
		if app == nil {
			apperrors.NotFoundError(ctx.Ctx, "Application not found", &ctx.DeviceID)
			return
		}
		appEmail = &app.Email
	}
	if appEmail == nil {
		appEmail = utils.GetStringPointer(ctx.GetStringContextData("WorkspaceEmail"))
	}
	link, err := payments.PaymentProcessor.GeneratePaymentLink(*appEmail, map[string]any{
		"workspaceID": ctx.GetStringContextData("WorkspaceID"),
		"appID":       ctx.Body.AppID,
		"reverse":     true,
	}, 500_00, []payment_types.PaymentChannel{payment_types.Card, payment_types.DirectDebit})
	if err != nil {
		apperrors.ExternalDependencyError(ctx.Ctx, "Paystack", "500", err, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "link generated", link.Link, nil, nil, &ctx.DeviceID)
}

func GeneratePaymentLink(ctx *interfaces.ApplicationContext[dto.GeneratePaymentLinkDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	appRepo := repository.ApplicationRepo()
	application, err := appRepo.FindByID(ctx.Body.AppID)
	if err != nil {
		logger.Error("an error occured while fetching app to generate payment link", logger.LoggerOptions{
			Key:  "err",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, nil, nil, ctx.DeviceID)
		return
	}
	if application == nil {
		apperrors.NotFoundError(ctx.Ctx, "Application not found", &ctx.DeviceID)
		return
	}
	subscriptionRepo := repository.SubscriptionPlanRepo()
	newSubscription, err := subscriptionRepo.FindByID(ctx.Body.PlanID)
	if err != nil {
		logger.Error("an error occured while fetching subscription", logger.LoggerOptions{
			Key:  "err",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "id",
			Data: ctx.Body.PlanID,
		})
		apperrors.UnknownError(ctx.Ctx, nil, nil, ctx.DeviceID)
		return
	}
	if newSubscription == nil {
		apperrors.NotFoundError(ctx.Ctx, "Invalid Subscription ID provided", &ctx.DeviceID)
		return
	}
	if newSubscription.Name == entities.Free || newSubscription.MonthlyPrice == 0 || newSubscription.AnnualPrice == 0 {
		apperrors.ClientError(ctx.Ctx, "You do not have to pay to be on the free plan", nil, nil, ctx.DeviceID)
		return
	}
	activeSubscriptionRepo := repository.ActiveSubscriptionRepo()
	activeSub, err := activeSubscriptionRepo.FindOneByFilter(map[string]interface{}{
		"appID": ctx.Body.AppID,
	})
	if err != nil {
		logger.Error("an error occured while fetching active subscription", logger.LoggerOptions{
			Key:  "err",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "id",
			Data: ctx.Body.PlanID,
		})
		apperrors.UnknownError(ctx.Ctx, nil, nil, ctx.DeviceID)
		return
	}
	var activeSubAmount uint32 = 0
	if activeSub != nil {
		subscriptionRepo := repository.SubscriptionPlanRepo()
		subscription, err := subscriptionRepo.FindByID(activeSub.ActiveSubID)
		if err != nil {
			logger.Error("an error occured while fetching subscription", logger.LoggerOptions{
				Key:  "err",
				Data: err,
			}, logger.LoggerOptions{
				Key:  "id",
				Data: ctx.Body.PlanID,
			})
			apperrors.UnknownError(ctx.Ctx, nil, nil, ctx.DeviceID)
			return
		}
		if subscription == nil {
			apperrors.NotFoundError(ctx.Ctx, "Invalid Subscription ID provided", &ctx.DeviceID)
			return
		}
		if activeSub.ActiveSubName == "Premium" && newSubscription.Name == "Essential" {
			if ctx.Body.Frequency == entities.Monthly {
				apperrors.ClientError(ctx.Ctx, "You do not need to pay to downgrade to an Essential subscription", nil, nil, ctx.DeviceID)
				return
			}
		}

		if activeSub.Interval == entities.Annually && ctx.Body.Frequency == entities.Monthly {
			apperrors.ClientError(ctx.Ctx, "You do not need to pay to switch to a monthly plan", nil, nil, ctx.DeviceID)
			return
		}

		if activeSub.Interval == ctx.Body.Frequency && ctx.Body.PlanID == activeSub.ActiveSubID {
			apperrors.ClientError(ctx.Ctx, "You are already on this plan", nil, nil, ctx.DeviceID)
			return
		}
		var subStartAt time.Time
		if activeSub.Interval == entities.Monthly {
			subStartAt = activeSub.ExpiresOn.AddDate(0, 0, -30)
		} else {
			subStartAt = activeSub.ExpiresOn.AddDate(0, 0, -365)
		}
		today := time.Now().UTC()
		daysElapsed := math.Ceil(-today.Sub(subStartAt).Hours() / 24)

		if activeSub.Interval == entities.Monthly {
			activeSubAmount = subscription.MonthlyPrice - ((subscription.MonthlyPrice / uint32(30)) * uint32(daysElapsed))
		} else {
			activeSubAmount = subscription.AnnualPrice - ((subscription.AnnualPrice / uint32(365)) * uint32(daysElapsed))
		}
	}
	var amount uint32
	switch ctx.Body.Frequency {
	case entities.Annually:
		amount = newSubscription.AnnualPrice
	case entities.Monthly:
		amount = newSubscription.MonthlyPrice
	default:
		apperrors.ClientError(ctx.Ctx, "Invalid frequency selected", nil, nil, ctx.DeviceID)
		return
	}
	link, err := payments.PaymentProcessor.GeneratePaymentLink(application.Email, map[string]any{
		"workspaceID": ctx.GetStringContextData("WorkspaceID"),
		"appID":       ctx.Body.AppID,
		"planID":      ctx.Body.PlanID,
		"frequency":   ctx.Body.Frequency,
		"autoRenew":   ctx.Body.AutoRenew,
	}, amount-activeSubAmount, []payment_types.PaymentChannel{payment_types.Card, payment_types.DirectDebit})
	if err != nil {
		apperrors.ExternalDependencyError(ctx.Ctx, "Paystack", "500", err, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "link generated", link.Link, nil, nil, &ctx.DeviceID)
}
