package util

import (
	"fmt"
	"log"

	"unibook-go/config"

	mail "github.com/xhit/go-simple-mail/v2"
)

func SendOtpEmail(cfg *config.Config, userEmail string, otp string) error {
	server := mail.NewSMTPClient()
	server.Host = cfg.SMTPHost
	server.Port = cfg.SMTPPort
	server.Username = cfg.SMTPUser
	server.Password = cfg.SMTPPass
	server.Encryption = mail.EncryptionTLS

	smtpClient, err := server.Connect()
	if err != nil {
		log.Printf("Failed to connect to SMTP server: %v", err)
		return err
	}

	email := mail.NewMSG()
	email.SetFrom(fmt.Sprintf("Unibook <%s>", cfg.EmailFrom)).
		AddTo(userEmail).
		SetSubject("Your Unibook Verification Code")

	htmlBody := fmt.Sprintf(`
      <div style="background-color: #ffffff; color: #000000; font-family: Arial, sans-serif; padding: 20px; text-align: center;">
        <h2 style="color: #000000;">Your Verification Code</h2>
        <p style="color: #333333;">Please use the following code to complete your registration.</p>
        <div style="font-size: 36px; font-weight: bold; letter-spacing: 8px; margin: 20px 0; color: #000000;">
          %s
        </div>
        <p style="color: #555555; font-size: 12px;">This code will expire in 10 minutes.</p>
      </div>`, otp)
	email.SetBody(mail.TextHTML, htmlBody)

	if err := email.Send(smtpClient); err != nil {
		log.Printf("Failed to send OTP email to %s: %v", userEmail, err)
		return err
	}

	log.Printf("Successfully sent OTP email to %s", userEmail)
	return nil
}
