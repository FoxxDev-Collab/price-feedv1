package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/foxxcyber/price-feed/internal/config"
	"github.com/foxxcyber/price-feed/internal/database"
)

// EmailService handles sending emails via SMTP
type EmailService struct {
	db            *database.DB
	cfg           *config.Config
	encryptionKey []byte
}

// NewEmailService creates a new email service instance
func NewEmailService(db *database.DB, cfg *config.Config) *EmailService {
	return &EmailService{
		db:            db,
		cfg:           cfg,
		encryptionKey: DeriveEncryptionKey(cfg.JWTSecret),
	}
}

// getSMTPConfig retrieves SMTP configuration from database
func (s *EmailService) getSMTPConfig(ctx context.Context) (*database.SMTPConfig, error) {
	return s.db.GetSMTPConfig(ctx, s.encryptionKey)
}

// IsConfigured returns true if SMTP is properly configured
func (s *EmailService) IsConfigured() bool {
	ctx := context.Background()
	smtpCfg, err := s.getSMTPConfig(ctx)
	if err != nil {
		return false
	}
	return smtpCfg.Enabled && smtpCfg.Host != "" && smtpCfg.FromAddr != ""
}

// IsConfiguredWithContext checks if SMTP is configured using provided context
func (s *EmailService) IsConfiguredWithContext(ctx context.Context) bool {
	smtpCfg, err := s.getSMTPConfig(ctx)
	if err != nil {
		return false
	}
	return smtpCfg.Enabled && smtpCfg.Host != "" && smtpCfg.FromAddr != ""
}

// SendEmail sends an email using SMTP
func (s *EmailService) SendEmail(to, subject, htmlBody, textBody string) error {
	ctx := context.Background()
	smtpCfg, err := s.getSMTPConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get SMTP config: %w", err)
	}

	if !smtpCfg.Enabled || smtpCfg.Host == "" {
		return fmt.Errorf("SMTP is not configured")
	}

	return s.sendMail(smtpCfg, []string{to}, subject, htmlBody, textBody)
}

// SendEmailToMultiple sends an email to multiple recipients
func (s *EmailService) SendEmailToMultiple(to []string, subject, htmlBody, textBody string) error {
	ctx := context.Background()
	smtpCfg, err := s.getSMTPConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get SMTP config: %w", err)
	}

	if !smtpCfg.Enabled || smtpCfg.Host == "" {
		return fmt.Errorf("SMTP is not configured")
	}

	return s.sendMail(smtpCfg, to, subject, htmlBody, textBody)
}

// SendTestEmail sends a test email to verify SMTP configuration
func (s *EmailService) SendTestEmail(to string) error {
	ctx := context.Background()
	smtpCfg, err := s.getSMTPConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get SMTP config: %w", err)
	}

	subject := "PriceFeed - Test Email"
	htmlBody := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9fafb; padding: 30px; border: 1px solid #e5e7eb; border-top: none; border-radius: 0 0 8px 8px; }
        .success { background: #d1fae5; border: 1px solid #10b981; color: #065f46; padding: 15px; border-radius: 6px; margin: 20px 0; }
        .footer { text-align: center; color: #6b7280; font-size: 12px; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1 style="margin: 0;">🎉 PriceFeed</h1>
            <p style="margin: 10px 0 0;">Email Configuration Test</p>
        </div>
        <div class="content">
            <div class="success">
                <strong>✅ Success!</strong> Your SMTP configuration is working correctly.
            </div>
            <p>This is a test email from your PriceFeed application. If you're receiving this message, your email settings are configured properly.</p>
            <p><strong>SMTP Settings Used:</strong></p>
            <ul>
                <li>Host: ` + smtpCfg.Host + `</li>
                <li>Port: ` + fmt.Sprintf("%d", smtpCfg.Port) + `</li>
                <li>From: ` + smtpCfg.FromName + ` &lt;` + smtpCfg.FromAddr + `&gt;</li>
            </ul>
            <p>You can now use email features like:</p>
            <ul>
                <li>User registration welcome emails</li>
                <li>Password reset emails</li>
                <li>Email verification</li>
                <li>Price alert notifications</li>
            </ul>
        </div>
        <div class="footer">
            <p>This email was sent from PriceFeed Admin Panel</p>
        </div>
    </div>
</body>
</html>`

	textBody := `PriceFeed - Test Email

Success! Your SMTP configuration is working correctly.

This is a test email from your PriceFeed application. If you're receiving this message, your email settings are configured properly.

SMTP Settings Used:
- Host: ` + smtpCfg.Host + `
- Port: ` + fmt.Sprintf("%d", smtpCfg.Port) + `
- From: ` + smtpCfg.FromName + ` <` + smtpCfg.FromAddr + `>

You can now use email features like:
- User registration welcome emails
- Password reset emails
- Email verification
- Price alert notifications

This email was sent from PriceFeed Admin Panel`

	return s.sendMail(smtpCfg, []string{to}, subject, htmlBody, textBody)
}

// SendWelcomeEmail sends a welcome email to a new user
func (s *EmailService) SendWelcomeEmail(to, username string) error {
	subject := "Welcome to PriceFeed!"
	htmlBody := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9fafb; padding: 30px; border: 1px solid #e5e7eb; border-top: none; border-radius: 0 0 8px 8px; }
        .btn { display: inline-block; background: #667eea; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin: 20px 0; }
        .footer { text-align: center; color: #6b7280; font-size: 12px; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1 style="margin: 0;">Welcome to PriceFeed!</h1>
        </div>
        <div class="content">
            <p>Hi ` + username + `,</p>
            <p>Thanks for joining PriceFeed! You're now part of a community-driven platform helping everyone find the best grocery prices.</p>
            <p>Here's what you can do:</p>
            <ul>
                <li>🔍 Search and compare prices across stores</li>
                <li>📝 Create shopping lists</li>
                <li>💰 Submit prices to help others save</li>
                <li>⭐ Earn reputation points for contributions</li>
            </ul>
            <p>Start exploring now!</p>
        </div>
        <div class="footer">
            <p>© PriceFeed - Community-driven grocery price comparison</p>
        </div>
    </div>
</body>
</html>`

	textBody := `Welcome to PriceFeed!

Hi ` + username + `,

Thanks for joining PriceFeed! You're now part of a community-driven platform helping everyone find the best grocery prices.

Here's what you can do:
- Search and compare prices across stores
- Create shopping lists
- Submit prices to help others save
- Earn reputation points for contributions

Start exploring now!

© PriceFeed - Community-driven grocery price comparison`

	return s.SendEmail(to, subject, htmlBody, textBody)
}

// SendEmailVerificationEmail sends an email verification email
func (s *EmailService) SendEmailVerificationEmail(to, verifyToken string, verifyURL string) error {
	subject := "Verify Your PriceFeed Email"

	fullVerifyURL := verifyURL + "?token=" + verifyToken

	htmlBody := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9fafb; padding: 30px; border: 1px solid #e5e7eb; border-top: none; border-radius: 0 0 8px 8px; }
        .btn { display: inline-block; background: #667eea; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin: 20px 0; }
        .info { background: #e0f2fe; border: 1px solid #0ea5e9; color: #0c4a6e; padding: 15px; border-radius: 6px; margin: 20px 0; }
        .footer { text-align: center; color: #6b7280; font-size: 12px; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1 style="margin: 0;">Verify Your Email</h1>
        </div>
        <div class="content">
            <p>Thanks for signing up for PriceFeed! Please verify your email address to complete your registration.</p>
            <p>Click the button below to verify your email:</p>
            <p style="text-align: center;">
                <a href="` + fullVerifyURL + `" class="btn">Verify Email</a>
            </p>
            <div class="info">
                <strong>ℹ️ Note:</strong> This link will expire in 24 hours. If you didn't create an account, you can safely ignore this email.
            </div>
            <p>If the button doesn't work, copy and paste this link into your browser:</p>
            <p style="word-break: break-all; color: #6b7280;">` + fullVerifyURL + `</p>
        </div>
        <div class="footer">
            <p>© PriceFeed - Community-driven grocery price comparison</p>
        </div>
    </div>
</body>
</html>`

	textBody := `Verify Your Email

Thanks for signing up for PriceFeed! Please verify your email address to complete your registration.

Click the link below to verify your email:
` + fullVerifyURL + `

Note: This link will expire in 24 hours. If you didn't create an account, you can safely ignore this email.

© PriceFeed - Community-driven grocery price comparison`

	return s.SendEmail(to, subject, htmlBody, textBody)
}

// SendPasswordResetEmail sends a password reset email
func (s *EmailService) SendPasswordResetEmail(to, resetToken string, resetURL string) error {
	subject := "Reset Your PriceFeed Password"

	fullResetURL := resetURL + "?token=" + resetToken

	htmlBody := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9fafb; padding: 30px; border: 1px solid #e5e7eb; border-top: none; border-radius: 0 0 8px 8px; }
        .btn { display: inline-block; background: #667eea; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin: 20px 0; }
        .warning { background: #fef3c7; border: 1px solid #f59e0b; color: #92400e; padding: 15px; border-radius: 6px; margin: 20px 0; }
        .footer { text-align: center; color: #6b7280; font-size: 12px; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1 style="margin: 0;">Password Reset</h1>
        </div>
        <div class="content">
            <p>You requested a password reset for your PriceFeed account.</p>
            <p>Click the button below to reset your password:</p>
            <p style="text-align: center;">
                <a href="` + fullResetURL + `" class="btn">Reset Password</a>
            </p>
            <div class="warning">
                <strong>⚠️ Important:</strong> This link will expire in 1 hour. If you didn't request this reset, please ignore this email.
            </div>
            <p>If the button doesn't work, copy and paste this link into your browser:</p>
            <p style="word-break: break-all; color: #6b7280;">` + fullResetURL + `</p>
        </div>
        <div class="footer">
            <p>© PriceFeed - Community-driven grocery price comparison</p>
        </div>
    </div>
</body>
</html>`

	textBody := `Password Reset Request

You requested a password reset for your PriceFeed account.

Click the link below to reset your password:
` + fullResetURL + `

Important: This link will expire in 1 hour. If you didn't request this reset, please ignore this email.

© PriceFeed - Community-driven grocery price comparison`

	return s.SendEmail(to, subject, htmlBody, textBody)
}

// sendMail is the internal method that handles SMTP communication
func (s *EmailService) sendMail(smtpCfg *database.SMTPConfig, to []string, subject, htmlBody, textBody string) error {
	// Build the email headers and body
	boundary := "boundary-pricefeed-email-12345"

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s <%s>\r\n", smtpCfg.FromName, smtpCfg.FromAddr))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
	msg.WriteString("\r\n")

	// Plain text part
	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	msg.WriteString("Content-Transfer-Encoding: 7bit\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(textBody)
	msg.WriteString("\r\n")

	// HTML part
	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	msg.WriteString("Content-Transfer-Encoding: 7bit\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)
	msg.WriteString("\r\n")

	msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", smtpCfg.Host, smtpCfg.Port)

	// Create authentication if credentials provided
	var auth smtp.Auth
	if smtpCfg.User != "" && smtpCfg.Password != "" {
		auth = smtp.PlainAuth("", smtpCfg.User, smtpCfg.Password, smtpCfg.Host)
	}

	// For ports 465, use implicit TLS
	if smtpCfg.Port == 465 {
		return s.sendMailWithTLS(smtpCfg, addr, auth, to, msg.String())
	}

	// For other ports (587, 25), use STARTTLS
	return s.sendMailWithSTARTTLS(smtpCfg, addr, auth, to, msg.String())
}

// sendMailWithTLS sends mail using implicit TLS (port 465)
func (s *EmailService) sendMailWithTLS(smtpCfg *database.SMTPConfig, addr string, auth smtp.Auth, to []string, msg string) error {
	tlsConfig := &tls.Config{
		ServerName: smtpCfg.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, smtpCfg.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	if err = client.Mail(smtpCfg.FromAddr); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data writer: %w", err)
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return client.Quit()
}

// sendMailWithSTARTTLS sends mail using STARTTLS (ports 587, 25)
func (s *EmailService) sendMailWithSTARTTLS(smtpCfg *database.SMTPConfig, addr string, auth smtp.Auth, to []string, msg string) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	// Try STARTTLS if available
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: smtpCfg.Host,
		}
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS failed: %w", err)
		}
	}

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	if err = client.Mail(smtpCfg.FromAddr); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data writer: %w", err)
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return client.Quit()
}

// GetConfig returns the current SMTP configuration (with password masked)
func (s *EmailService) GetConfig() map[string]interface{} {
	ctx := context.Background()
	smtpCfg, err := s.getSMTPConfig(ctx)
	if err != nil {
		return map[string]interface{}{
			"error":      err.Error(),
			"configured": false,
		}
	}

	passwordMask := ""
	if smtpCfg.Password != "" {
		passwordMask = "••••••••"
	}

	return map[string]interface{}{
		"host":       smtpCfg.Host,
		"port":       smtpCfg.Port,
		"user":       smtpCfg.User,
		"password":   passwordMask,
		"fromAddr":   smtpCfg.FromAddr,
		"fromName":   smtpCfg.FromName,
		"enabled":    smtpCfg.Enabled,
		"configured": smtpCfg.Enabled && smtpCfg.Host != "" && smtpCfg.FromAddr != "",
	}
}
