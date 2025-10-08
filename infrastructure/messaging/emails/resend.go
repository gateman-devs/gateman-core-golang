package emails

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"gateman.io/infrastructure/logger"
	"github.com/resend/resend-go/v2"
)

var rsdir, _ = os.Getwd()

type ResendService struct {
}

func (rs *ResendService) SendEmail(toEmail string, subject string, templateName string, opts interface{}) bool {
	apiKey := os.Getenv("RESEND_API_KEY")

	client := resend.NewClient(apiKey)

	html := rs.loadTemplates(templateName, opts)
	if html == nil {
		logger.Error("failed to load email template", logger.LoggerOptions{
			Key:  "templateName",
			Data: templateName,
		}, logger.LoggerOptions{
			Key:  "toEmail",
			Data: toEmail,
		})
		return false
	}

	params := &resend.SendEmailRequest{
		From:    os.Getenv("RESEND_DEFAULT_EMAIL"),
		To:      []string{toEmail},
		Subject: subject,
		Html:    *html,
	}

	_, err := client.Emails.Send(params)
	if err != nil {
		logger.Error("an error occured while trying to send email using resend service", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "toEmail",
			Data: toEmail,
		}, logger.LoggerOptions{
			Key:  "templateName",
			Data: templateName,
		})
		return false
	}
	logger.Info(fmt.Sprintf("successfully sent email to %s", toEmail), logger.LoggerOptions{
		Key:  "templateName",
		Data: templateName,
	}, logger.LoggerOptions{
		Key:  "service",
		Data: "resend",
	})
	return true
}

func (rs *ResendService) loadTemplates(templateName string, opts interface{}) *string {
	var buffer bytes.Buffer
	templatePath := filepath.Join(rsdir, "infrastructure", "messaging", "emails", "templates", templateName+".html")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		logger.Error("failed to parse email template", logger.LoggerOptions{
			Key:  "templateName",
			Data: templateName,
		}, logger.LoggerOptions{
			Key:  "templatePath",
			Data: templatePath,
		}, logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil
	}
	err = tmpl.Execute(&buffer, opts)
	if err != nil {
		logger.Error("failed to execute email template", logger.LoggerOptions{
			Key:  "templateName",
			Data: templateName,
		}, logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil
	}
	templateString := buffer.String()
	return &templateString
}
