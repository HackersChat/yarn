// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/go-mail/mail"
	log "github.com/sirupsen/logrus"
)

var (
	ErrSendingEmail = errors.New("error: unable to send email")

	magicLinkAuthEmailTemplate = template.Must(template.New("email").Parse(`Hello {{ .Username }},

You have requested to login to your Yarn.social account on {{ .Pod }} via email. If this was **NOT** initiated by you, please ignore this email and contact support.

To login to your account, please visit the following link:
{{ .BaseURL }}/magiclinkauth?token={{ .Token }}

Kind regards,

{{ .Pod }} Support
`))

	passwordResetEmailTemplate = template.Must(template.New("email").Parse(`Hello {{ .Username }},

You have requested to have your password on {{ .Pod }} reset for your account.

**IMPORTANT:** If this was **NOT** initiated by you, please ignore this email and contact support!

To reset your password, please visit the following link:

{{ .BaseURL }}/newPassword?token={{ .Token }}

Kind regards,

{{ .Pod }} Support
`))

	supportRequestEmailTemplate = template.Must(template.New("email").Parse(`Hello {{ .AdminUser }},

{{ .Name }} <{{ .Email }} from {{ .Pod }} has sent the following support request:

Subject: {{ .Subject }}
{{ .Message }}

Kind regards,

{{ .Pod }} Support
`))

	candidatesForDeletionEmailTemplate = template.Must(template.New("email").Parse(`Hello {{ .AdminUser }},

The following top 10 users are candidates for deletion as they have either never posted, updated their profile or follow any feeds. Their usernames on {{ .Pod }} are shown below with their scores. The higher the score the more likely the user has never used their account.

{{ range $candidate := .Candidates }}
{{ $candidate.Username }}
Score: {{ $candidate.Score }}
{{ $.BaseURL }}/user/{{ $candidate.Username }}
{{ end }}

To delete any of these users visit:
{{ .BaseURL }}/manage/users

Kind regards,

{{ .Pod }} Support
`))

	reportAbuseEmailTemplate = template.Must(template.New("email").Parse(`Hello {{ .AdminUser }},

{{ .Name }} {{ .Email }} from {{ .Pod }} has sent the following abuse report:

Category: {{ .Category }}
{{ .Message }}

The offending user/feed in question is:

- Nick: {{ .Nick }}
- URL: {{ .URL }}

Kind regards,

{{ .Pod }} Support
`))

	newUserEmailTemplate = template.Must(template.New("email").Parse(`Hello {{ .AdminUser }},

This message is to let you know that a new user has joined your {{ .Pod }} pod!

Nick: {{ .Username }}
Feed: {{ .BaseURL }}/user/{{ .Username }}

Kind regards,

{{ .Pod }} Support
`))
)

type MagicLinkAuthContext struct {
	Pod     string
	BaseURL string

	Token    string
	Username string
}

type PasswordResetEmailContext struct {
	Pod     string
	BaseURL string

	Token    string
	Username string
}

type SupportRequestEmailContext struct {
	Pod       string
	AdminUser string

	Name    string
	Email   string
	Subject string
	Message string
}

type DeletionCandidate struct {
	Username string
	Score    int
}

type CandidatesByScore []DeletionCandidate

func (c CandidatesByScore) Len() int           { return len(c) }
func (c CandidatesByScore) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c CandidatesByScore) Less(i, j int) bool { return c[i].Score < c[j].Score }

type CandidatesForDeletionEmailContext struct {
	Pod       string
	BaseURL   string
	AdminUser string

	Candidates []DeletionCandidate
}

type ReportAbuseEmailContext struct {
	Pod       string
	AdminUser string

	Nick string
	URL  string

	Name     string
	Email    string
	Category string
	Message  string
}

type NewUserEmailContext struct {
	Pod       string
	BaseURL   string
	AdminUser string

	Username string
}

// Indent indents a block of text with an indent string
func Indent(text, indent string) string {
	result := ""
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		result += indent + scanner.Text() + "\n"
	}
	return result
}

func SendEmail(conf *Config, recipients []string, replyTo, subject string, body string) error {
	if conf.SMTPHost == "" || conf.SMTPHost == DefaultSMTPHost ||
		conf.SMTPPort == 0 ||
		conf.SMTPUser == "" || conf.SMTPUser == DefaultSMTPUser ||
		conf.SMTPPass == "" || conf.SMTPPass == DefaultSMTPPass ||
		conf.SMTPFrom == "" || conf.SMTPFrom == DefaultSMTPFrom {
		log.Warn("sending emails disabled in configuration")
		return nil
	}

	m := mail.NewMessage()
	m.SetHeader("From", conf.SMTPFrom)
	m.SetHeader("To", recipients...)
	m.SetHeader("Reply-To", replyTo)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	d := mail.NewDialer(conf.SMTPHost, conf.SMTPPort, conf.SMTPUser, conf.SMTPPass)

	err := d.DialAndSend(m)
	if err != nil {
		log.WithError(err).Error("SendEmail() failed")
		return ErrSendingEmail
	}

	return nil
}

func SendMagicLinkAuthEmail(conf *Config, user *User, email, token string) error {
	recipients := []string{email}
	subject := fmt.Sprintf(
		"[%s]: Login via Email for %s",
		conf.Name, user.Username,
	)
	ctx := MagicLinkAuthContext{
		Pod:     conf.Name,
		BaseURL: conf.BaseURL,

		Token:    token,
		Username: user.Username,
	}

	buf := &bytes.Buffer{}
	if err := magicLinkAuthEmailTemplate.Execute(buf, ctx); err != nil {
		log.WithError(err).Error("error rendering email template")
		return err
	}

	if err := SendEmail(conf, recipients, conf.SMTPFrom, subject, buf.String()); err != nil {
		log.WithError(err).Errorf("error sending new token to %s", recipients[0])
		return err
	}

	return nil
}

func SendPasswordResetEmail(conf *Config, user *User, email, token string) error {
	recipients := []string{email}
	subject := fmt.Sprintf(
		"[%s]: Password Reset Request for %s",
		conf.Name, user.Username,
	)
	ctx := PasswordResetEmailContext{
		Pod:     conf.Name,
		BaseURL: conf.BaseURL,

		Token:    token,
		Username: user.Username,
	}

	buf := &bytes.Buffer{}
	if err := passwordResetEmailTemplate.Execute(buf, ctx); err != nil {
		log.WithError(err).Error("error rendering email template")
		return err
	}

	if err := SendEmail(conf, recipients, conf.SMTPFrom, subject, buf.String()); err != nil {
		log.WithError(err).Errorf("error sending new token to %s", recipients[0])
		return err
	}

	return nil
}

func SendSupportRequestEmail(conf *Config, name, email, subject, message string) error {
	recipients := []string{conf.AdminEmail, email}
	emailSubject := fmt.Sprintf(
		"[%s Support Request]: %s",
		conf.Name, subject,
	)
	ctx := SupportRequestEmailContext{
		Pod:       conf.Name,
		AdminUser: conf.AdminUser,

		Name:    name,
		Email:   email,
		Subject: subject,
		Message: Indent(message, "> "),
	}

	buf := &bytes.Buffer{}
	if err := supportRequestEmailTemplate.Execute(buf, ctx); err != nil {
		log.WithError(err).Error("error rendering email template")
		return err
	}

	if err := SendEmail(conf, recipients, email, emailSubject, buf.String()); err != nil {
		log.WithError(err).Errorf("error sending support request to %s", recipients[0])
		return err
	}

	return nil
}

func SendCandidatesForDeletionEmail(conf *Config, candidates []DeletionCandidate) error {
	recipients := []string{conf.AdminEmail}
	emailSubject := fmt.Sprintf(
		"[%s Candidates for Deletion]: %d users",
		conf.Name, len(candidates),
	)
	ctx := CandidatesForDeletionEmailContext{
		Pod:       conf.Name,
		BaseURL:   conf.BaseURL,
		AdminUser: conf.AdminUser,

		Candidates: candidates,
	}

	buf := &bytes.Buffer{}
	if err := candidatesForDeletionEmailTemplate.Execute(buf, ctx); err != nil {
		log.WithError(err).Error("error rendering email template")
		return err
	}

	if err := SendEmail(conf, recipients, conf.SMTPFrom, emailSubject, buf.String()); err != nil {
		log.WithError(err).Errorf("error sending candidates for deletoin email to %s", recipients[0])
		return err
	}

	return nil
}

func SendReportAbuseEmail(conf *Config, nick, url, name, email, category, message string) error {
	recipients := []string{conf.AdminEmail, email}
	emailSubject := fmt.Sprintf(
		"[%s Report Abuse]: %s",
		conf.Name, category,
	)
	ctx := ReportAbuseEmailContext{
		Pod:       conf.Name,
		AdminUser: conf.AdminUser,

		Nick: nick,
		URL:  url,

		Name:     name,
		Email:    email,
		Category: category,
		Message:  Indent(message, "> "),
	}

	buf := &bytes.Buffer{}
	if err := reportAbuseEmailTemplate.Execute(buf, ctx); err != nil {
		log.WithError(err).Error("error rendering email template")
		return err
	}

	if err := SendEmail(conf, recipients, email, emailSubject, buf.String()); err != nil {
		log.WithError(err).Errorf("error sending report abuse to %s", recipients[0])
		return err
	}

	return nil
}

func SendNewUserEmail(conf *Config, username string) error {
	recipients := []string{conf.AdminEmail}
	emailSubject := fmt.Sprintf(
		"[%s New User]: %s",
		conf.Name, username,
	)
	ctx := NewUserEmailContext{
		Pod:       conf.Name,
		BaseURL:   conf.BaseURL,
		AdminUser: conf.AdminUser,

		Username: username,
	}

	buf := &bytes.Buffer{}
	if err := newUserEmailTemplate.Execute(buf, ctx); err != nil {
		log.WithError(err).Error("error rendering email template")
		return err
	}

	if err := SendEmail(conf, recipients, conf.SMTPFrom, emailSubject, buf.String()); err != nil {
		log.WithError(err).Errorf("error sending notification to %s", recipients[0])
		return err
	}

	return nil
}
