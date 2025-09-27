// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestIndent(t *testing.T) {
	assert := assert.New(t)

	t.Run("Empty", func(t *testing.T) {
		text := Indent("", "> ")
		assert.Equal("", text)
	})

	t.Run("Single line", func(t *testing.T) {
		text := Indent("foo", "> ")
		assert.Equal("> foo\n", text)
	})

	t.Run("Multiple lines", func(t *testing.T) {
		text := Indent("foo\nbar\nbaz", "> ")
		assert.Equal("> foo\n> bar\n> baz\n", text)
	})
}

func TestSendEmail_whenDisabledInConfig_thenLogWarning(t *testing.T) {
	filledSMTPConfig := func(updateConfig func(config *Config)) func() *Config {
		return func() *Config {
			conf := &Config{
				SMTPHost: "smtp.example.com",
				SMTPPort: 587,
				SMTPUser: "user",
				SMTPPass: "hunter2",
				SMTPFrom: "from@example.com",
			}
			updateConfig(conf)
			return conf
		}
	}
	for _, tt := range []struct {
		name  string
		setup func() *Config
	}{
		{"default config", NewConfig},
		{"default SMTP host", filledSMTPConfig(func(conf *Config) { conf.SMTPHost = DefaultSMTPHost })},
		{"default SMTP port", filledSMTPConfig(func(conf *Config) { conf.SMTPPort = DefaultSMTPPort })},
		{"default SMTP user", filledSMTPConfig(func(conf *Config) { conf.SMTPUser = DefaultSMTPUser })},
		{"default SMTP pass", filledSMTPConfig(func(conf *Config) { conf.SMTPPass = DefaultSMTPPass })},
		{"default SMTP from", filledSMTPConfig(func(conf *Config) { conf.SMTPFrom = DefaultSMTPFrom })},
		{"empty SMTP host", filledSMTPConfig(func(conf *Config) { conf.SMTPHost = "" })},
		{"empty SMTP port", filledSMTPConfig(func(conf *Config) { conf.SMTPPort = 0 })},
		{"empty SMTP user", filledSMTPConfig(func(conf *Config) { conf.SMTPUser = "" })},
		{"empty SMTP pass", filledSMTPConfig(func(conf *Config) { conf.SMTPPass = "" })},
		{"empty SMTP from", filledSMTPConfig(func(conf *Config) { conf.SMTPFrom = "" })},
	} {
		t.Run(tt.name, func(t *testing.T) {
			conf := tt.setup()
			hook := test.NewGlobal()

			assert.NoError(t, SendEmail(conf, []string{"recipient"}, "reply to", "subject", "body"))

			if assert.Len(t, hook.Entries, 1) {
				entry := hook.Entries[0]
				assert.Equal(t, log.WarnLevel, entry.Level)
				assert.Equal(t, "sending emails disabled in configuration", entry.Message)
			}
		})
	}
}
