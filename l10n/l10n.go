/*
Copyright 2024 Kirill Belousov <cyrmax@internet.ru>
Use of this source code is governed by a MIT license that can be found in the LICENSE file.
*/

// Package l10n is a simple localization package which supports switching between several locales on every localization request.
package l10n

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/BurntSushi/toml"
)

// L10nBundle represents a localization messages bundle for one language.
type L10nBundle struct {
	LocaleCode        string            `toml:"locale-code"`
	LocaleDisplayName string            `toml:"display-name"`
	Messages          map[string]string `toml:"messages"`
}

func (b *L10nBundle) Get(key string) string {
	if text, ok := b.Messages[key]; ok {
		return text
	}
	return key
}

func NewBundle() *L10nBundle {
	return &L10nBundle{LocaleCode: "empty", LocaleDisplayName: "Inexistent language", Messages: make(map[string]string)}
}

func NewBundleFromString(str string) (bundle *L10nBundle, err error) {
	_, err = toml.Decode(str, bundle)
	if err != nil {
		err = errors.Wrap(err, "unable to load l10n bundle")
	}
	return
}

func NewBundleFromFile(filePath string) (bundle *L10nBundle, err error) {
	bundle = &L10nBundle{}
	_, err = toml.DecodeFile(filePath, bundle)
	if err != nil {
		err = errors.Wrap(err, "unable to load l10n bundle")
	}
	return
}

type Localizer struct {
	fallbackLocale string
	bundles        map[string]*L10nBundle
}

func NewLocalizer(fallbackLocale string) *Localizer {
	return &Localizer{fallbackLocale: fallbackLocale, bundles: make(map[string]*L10nBundle)}
}

func (l *Localizer) AddBundle(bundle *L10nBundle) error {
	if _, ok := l.bundles[bundle.LocaleCode]; ok {
		return errors.New("bundle already added	")
	}
	l.bundles[bundle.LocaleCode] = bundle
	return nil
}

func (l *Localizer) Get(key string, lng string) string {
	if bundle, ok := l.bundles[lng]; ok {
		return bundle.Get(key)
	}
	if bundle, ok := l.bundles[l.fallbackLocale]; ok {
		return bundle.Get(key)
	}
	return key
}

func (l *Localizer) Getf(key string, lng string, args ...interface{}) string {
	return fmt.Sprintf(l.Get(key, lng), args...)
}
