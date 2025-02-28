package subscription

import (
	"gateman.io/application/repository"
	"gateman.io/entities"
)

func SeedSubscriptionData() {
	var subscriptionData []entities.SubscriptionPlan = []entities.SubscriptionPlan{
		{
			Name:         "Free",
			MonthlyPrice: 0,
			AnnualPrice:  0,
			Features:     []string{"10,000 Monthly Active Users", "Import users from your existing DB", "Passwordless authentication", "Custom validated signup form (max 3 fields)", "Access and Refresh token pair", "Basic attack protection"},
		}, {
			Name:         "Essential",
			MonthlyPrice: 87_000_00,
			AnnualPrice:  960_000_00,
			Features:     []string{"Everything in Free", "15,000 Monthly Active Users", "Access to verified user data if authorized", "Access to sensitive user data if authorized", "Custom validated signup form (unlimited fields)", "Gateman MFA Authentication option", "Customized sign up and sign in pages", "Custom Email Sender"},
		},
		{
			Name:         "Premium",
			MonthlyPrice: 300_000_00,
			AnnualPrice:  3_000_000_00,
			Features:     []string{"Everything in Essential", "25,000 Monthly Active Users", "2FA authentication option", "User activity log", "User data updates via webhooks"},
		},
	}
	subPlanRepo := repository.SubscriptionPlanRepo()
	seeded, _ := subPlanRepo.CountDocs(map[string]interface{}{})
	if seeded != 0 {
		return
	}
	subPlanRepo.CreateBulk(subscriptionData)
}
