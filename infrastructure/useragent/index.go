package useragent

import "github.com/mileusna/useragent"

func ParseUserAgent(userAgent string) *UserAgent {
	parsed := useragent.Parse(userAgent)
	return &UserAgent{
		Bot:       parsed.Bot,
		OS:        parsed.OS,
		OSVersion: parsed.VersionNoFull(),
		Device:    parsed.Device,
		Name:      parsed.Name,
	}
}
