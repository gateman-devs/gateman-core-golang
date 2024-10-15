package subscription

import (
	"os"

	"authone.usepolymer.co/application/repository"
	"authone.usepolymer.co/entities"
)

func SeedSubscriptionData() {
	var subscriptionData []entities.SubscriptionPlan = []entities.SubscriptionPlan{
		{
			Name:         "Free",
			MonthlyPrice: "₦85,000",
			AnnualPrice:  "₦960,000",
			Features:     []string{"2,500 Monthly Active Users", "Import users from your existing DB", "Passwordless authentication", "Custom validated signup form (max 3 fields)", "Google and GitHub SSO", "Access and Refresh token pair", "Basic attack protection"},
		}, {
			Name:         "Essential",
			MonthlyPrice: "₦85,000",
			AnnualPrice:  "₦960,000",
			AnnualURL:    os.Getenv("PAYSTACK_ESSENTIAL_ANNUAL_URL"),
			MonthlyURL:   os.Getenv("PAYSTACK_ESSENTIAL_MONTHLY_URL"),
			Features:     []string{"Everything in Free", "8000 Monthly Active Users", "Access to verified user data if authorized", "Access to sensitive user data if authorized", "Custom validated signup form (unlimited fields)", "Unlimited SSO options", "Passkey Authentication", "Customized sign up and sign in pages", "Custom Domain", "Custom Email", "Advanced attack protection"},
		},
		{
			Name:         "Premium",
			MonthlyPrice: "₦300,000",
			AnnualPrice:  "₦3,000,000",
			AnnualURL:    os.Getenv("PAYSTACK_PREMIUM_ANNUAL_URL"),
			MonthlyURL:   os.Getenv("PAYSTACK_PREMIUM_MONTHLY_URL"),
			Features:     []string{"Everything in Essential", "13,000 Monthly Active Users", "2FA authentication option", "User activity log", "User data updates via webhooks"},
		},
	}
	subPlanRepo := repository.SubscriptionPlanRepo()
	seeded, _ := subPlanRepo.CountDocs(map[string]interface{}{})
	if seeded != 0 {
		return
	}
	subPlanRepo.CreateBulk(subscriptionData)
}
