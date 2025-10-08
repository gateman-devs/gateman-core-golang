package useragent

import (
	"sync"

	"github.com/ua-parser/uap-go/uaparser"
)

var (
	parser *uaparser.Parser
)

func ParseUserAgent(userAgent string) *UserAgent {
	var initParser sync.Once
	initParser.Do(func() {
		parser = uaparser.NewFromSaved()
	})
	client := parser.Parse(userAgent)
	return &UserAgent{
		OS:        client.Os.ToString(),
		OSVersion: client.Os.ToVersionString(),
		Device:    client.Device.Model,
		Name:      client.Device.ToString(),
	}
}
