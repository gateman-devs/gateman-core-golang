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
			Features:     []string{"10,000 Monthly Active Users", "Import users from your existing DB (coming soon)", "Passwordless authentication", "No-code validated SignUp form", "User activity log 30 day retention (coming soon)", "Access and Refresh token pair"},
		}, {
			Name:         entities.Essential,
			MonthlyPrice: 87_000_00,
			AnnualPrice:  960_000_00,
			Features:     []string{"Everything in Free", "Unlimited Active Users", "Free Access to NIN, BVN, Voter ID and Drivers license data", "Access to verified user data if authorized", "Access to sensitive user data if authorized", "User activity log 60 day retention", "Pin protected accounts"},
		},
		{
			Name:         entities.Premium,
			MonthlyPrice: 300_000_00,
			AnnualPrice:  3_000_000_00,
			Features:     []string{"Everything in Essential", "MFA protected accounts", "Remove Gateman branding", "Custom email domain sender", "User activity log 90 day retention", "User data updates via webhooks"},
		},
	}
	subPlanRepo := repository.SubscriptionPlanRepo()
	seeded, _ := subPlanRepo.CountDocs(map[string]interface{}{})
	if seeded != 0 {
		return
	}
	subPlanRepo.CreateBulk(subscriptionData)
}
