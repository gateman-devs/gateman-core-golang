package subscription

import (
	"authone.usepolymer.co/application/repository"
	"authone.usepolymer.co/entities"
)

func SeedSubscriptionData() {
	var subscriptionData []entities.SubscriptionPlan = []entities.SubscriptionPlan{
		{
			Name:         "Free",
			MonthlyPrice: "₦85,000",
			AnnualPrice:  "₦960,000",
			Features:     []string{"Unlimited user sign up and sign in", "Import existing users from your database", "Passwordless sign in", "Custom validated sign up form with max 3 fields", "Google and GitHub SSO options only", "Access and Refresh token pair"},
		},
		{
			Name:         "Professional",
			MonthlyPrice: "₦300,000",
			AnnualPrice:  "3,000,000",
			Features:     []string{"Everything in free", "Access to verified user data if authorized", "Access to sensitive user data if authorized", "Custom validated sign up form with unlimited fields", "Passkey sign in", "Customized sign up and sign in pages", "Custom domain", "Custom email"},
		},
	}
	subPlanRepo := repository.SubscriptionPlanRepo()
	seeded, _ := subPlanRepo.CountDocs(map[string]interface{}{})
	if seeded != 0 {
		return
	}
	subPlanRepo.CreateBulk(subscriptionData)
}
