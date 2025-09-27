// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"encoding/json"
	"fmt"
	"strings"

	sync "github.com/sasha-s/go-deadlock"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type FeatureType int

const (
	// FeatureInvalid is the invalid feature (0)
	FeatureInvalid FeatureType = iota
	FeatureFoo
)

// Interface guards
var (
	_ yaml.Marshaler   = (*FeatureFlags)(nil)
	_ yaml.Unmarshaler = (*FeatureFlags)(nil)
	_ json.Marshaler   = (*FeatureFlags)(nil)
	_ json.Unmarshaler = (*FeatureFlags)(nil)
)

func (f FeatureType) String() string {
	switch f {
	case FeatureFoo:
		return "foo"
	default:
		return "invalid_feature"
	}
}

func AvailableFeatures() []string {
	features := []string{}
	for i := 1; i <= 99; i++ {
		feature := FeatureType(i)
		if feature.String() == FeatureInvalid.String() {
			break
		}
		features = append(features, feature.String())
	}
	return features
}

func IsFeatureEnabled(fs *FeatureFlags, name string) bool {
	feature, err := FeatureFromString(name)
	if err != nil {
		return false
	}
	return fs.IsEnabled(feature)
}

func FeatureFromString(s string) (FeatureType, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "foo":
		return FeatureFoo, nil
	default:
		fs := fmt.Sprintf("available features: %s", strings.Join(AvailableFeatures(), " "))
		return FeatureInvalid, fmt.Errorf("Error unrecognised feature: %s (%s)", s, fs)
	}
}

func FeaturesFromStrings(xs []string) ([]FeatureType, error) {
	var features []FeatureType

	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}

		f, err := FeatureFromString(x)
		if err != nil {
			return nil, err
		}
		features = append(features, f)
	}

	return features, nil
}

// FeatureFlags describes a set of Pods optional Features
// and whether they are enabled or disabled
type FeatureFlags struct {
	sync.RWMutex
	flags map[FeatureType]bool
}

func NewFeatureFlags() *FeatureFlags {
	return &FeatureFlags{flags: make(map[FeatureType]bool)}
}

func (f *FeatureFlags) AsStrings() []string {
	var vs []string
	f.RLock()
	for flag := range f.flags {
		vs = append(vs, flag.String())
	}
	f.RUnlock()
	return vs
}

func (f *FeatureFlags) String() string {
	return strings.Join(f.AsStrings(), " ")
}

func (f *FeatureFlags) Disable(feature FeatureType) {
	f.Lock()
	defer f.Unlock()

	f.flags[feature] = false
}

func (f *FeatureFlags) Reset() {
	f.Lock()
	defer f.Unlock()

	f.flags = make(map[FeatureType]bool)
}

func (f *FeatureFlags) DisableAll(features []FeatureType) {
	f.Lock()
	defer f.Unlock()

	for _, feature := range features {
		f.flags[feature] = false
	}
}

func (f *FeatureFlags) Enable(feature FeatureType) {
	f.Lock()
	defer f.Unlock()

	f.flags[feature] = true
}

func (f *FeatureFlags) EnableAll(features []FeatureType) {
	f.Lock()
	defer f.Unlock()

	for _, feature := range features {
		f.flags[feature] = true
	}
}

func (f *FeatureFlags) IsEnabled(feature FeatureType) bool {
	f.RLock()
	defer f.RUnlock()

	return f.flags[feature]
}

func (f *FeatureFlags) MarshalJSON() ([]byte, error) {
	var vs []string
	f.RLock()
	for flag := range f.flags {
		vs = append(vs, flag.String())
	}
	f.RUnlock()
	return json.Marshal(vs)
}

func (f *FeatureFlags) UnmarshalJSON(b []byte) error {
	var vs []string
	if err := json.Unmarshal(b, &vs); err != nil {
		return err
	}

	features, err := FeaturesFromStrings(vs)
	if err != nil {
		log.WithError(err).Warnf("error parsing features: %#v", vs)
		return nil
	}

	f.Lock()
	f.flags = make(map[FeatureType]bool)
	for _, feature := range features {
		f.flags[feature] = true
	}
	f.Unlock()

	return nil
}

func (f *FeatureFlags) MarshalYAML() (interface{}, error) {
	var vs []string
	f.RLock()
	for flag := range f.flags {
		vs = append(vs, flag.String())
	}
	f.RUnlock()
	return vs, nil
}

func (f *FeatureFlags) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var vs []string

	err := unmarshal(&vs)
	if err != nil {
		return err
	}

	features, err := FeaturesFromStrings(vs)
	if err != nil {
		log.WithError(err).Warnf("error parsing features: %#v", vs)
		return nil
	}

	f.Lock()
	f.flags = make(map[FeatureType]bool)
	for _, feature := range features {
		f.flags[feature] = true
	}
	f.Unlock()

	return nil
}

// WithEnabledFeatures enables the selected features
func WithEnabledFeatures(features []FeatureType) Option {
	return func(cfg *Config) error {
		cfg.Features.Reset()
		cfg.Features.EnableAll(features)
		return nil
	}
}
