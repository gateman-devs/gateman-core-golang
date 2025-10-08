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
			Features: []string{
				"Scale to 10,000 Monthly Active Users",
				"Seamless User Migration from Existing Databases (launching soon)",
				"Secure Passwordless Authentication",
				"Drag-and-Drop Validated SignUp Forms",
				"30-Day User Activity Analytics & Monitoring (launching soon)",
				"Enterprise-Grade Access & Refresh Token Security",
			},
		}, {
			Name:         entities.Essential,
			MonthlyPrice: 87_000_00,
			AnnualPrice:  960_000_00,
			Features: []string{
				"Everything in Free Plan",
				"Unlimited Monthly Active Users",
				"Instant Government ID Verification (NIN, BVN, Voter ID, Driver's License)",
				"Verified User Data Access with Compliance Controls",
				"Secure Sensitive Data Access with Authorization Protocols",
				"Extended 60-Day User Activity Analytics & Monitoring",
				"PIN-Protected Account Security",
			},
		},
		{
			Name:         entities.Premium,
			MonthlyPrice: 300_000_00,
			AnnualPrice:  3_000_000_00,
			Features: []string{
				"Everything in Essential Plan",
				"Advanced Multi-Factor Authentication (MFA)",
				"White-Label Solution - Remove Gateman Branding",
				"Custom Branded Email Domain Integration",
				"Comprehensive 90-Day User Activity Analytics & Monitoring",
				"Real-Time User Data Updates via Webhooks",
				"Unlimited Government ID Data Access & Verification",
			},
		},
	}
	subPlanRepo := repository.SubscriptionPlanRepo()
	seeded, _ := subPlanRepo.CountDocs(map[string]interface{}{})
	if seeded != 0 {
		return
	}
	subPlanRepo.CreateBulk(subscriptionData)
}
