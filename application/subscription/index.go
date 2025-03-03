package subscription

import (
	"gateman.io/application/repository"
	"gateman.io/entities"
)

func SeedSubscriptionData() {
	var subscriptionData []entities.SubscriptionPlan = []entities.SubscriptionPlan{
		{
			Name:         entities.Free,
			MonthlyPrice: 0,
			AnnualPrice:  0,
			Features:     []string{"10,000 Monthly Active Users", "Import users from your existing DB", "Passwordless authentication", "Custom validated signup form (max 3 fields)", "User activity log 30 day retention", "Access and Refresh token pair"},
		}, {
			Name:         entities.Essential,
			MonthlyPrice: 87_000_00,
			AnnualPrice:  960_000_00,
			Features:     []string{"Everything in Free", "Unlimited Active Users", "Access to verified user data if authorized", "Access to sensitive user data if authorized", "User activity log 60 day retention", "Custom validated signup form (unlimited fields)", "Pin protected accounts"},
		},
		{
			Name:         entities.Premium,
			MonthlyPrice: 300_000_00,
			AnnualPrice:  3_000_000_00,
			Features:     []string{"Everything in Essential", "Unlimited Active Users", "MFA protected accounts", "Remove Gateman branding", "Custom email domain sender", "User activity log 90 day retention", "User data updates via webhooks"},
		},
	}
	subPlanRepo := repository.SubscriptionPlanRepo()
	seeded, _ := subPlanRepo.CountDocs(map[string]interface{}{})
	if seeded != 0 {
		return
	}
	subPlanRepo.CreateBulk(subscriptionData)
}
