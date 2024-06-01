package emails

type EmailServiceType interface {
	SendEmail(toEmail string, subject string, templateName string, opts interface{}) bool
	loadTemplates(templateName string, opts interface{}) *string
}
