// Package i18n is the app-facing catalog API.
//
// The engine loads apps/<name>/locale/*.po during Setup. Translate with T / Tf
// (or HookContext.T in lifecycle hooks). Missing keys fall back to English, then the key.
package i18n

import (
	"errors"

	platform "kaizengo/internal/platform/i18n"
)

type TextDirection = platform.TextDirection

const (
	LTR TextDirection = platform.LTR
	RTL TextDirection = platform.RTL
)

type LocaleInfo = platform.LocaleInfo

// T translates key for the active locale, then en, then returns the key.
func T(key string) string {
	return platform.T(key)
}

// Tf is T with fmt.Sprintf-style formatting (%s, %d, …).
func Tf(key string, args ...any) string {
	return platform.Tf(key, args...)
}

// Error returns an error whose message is T(key).
func Error(key string) error {
	return errors.New(T(key))
}

// Errorf returns an error whose message is Tf(key, args...).
func Errorf(key string, args ...any) error {
	return errors.New(Tf(key, args...))
}

func Locale() string { return platform.Locale() }

func SetLocale(id string) { platform.SetLocale(id) }

func Dir() TextDirection { return platform.Dir() }

func Info(localeID string) LocaleInfo { return platform.Info(localeID) }

func ActiveInfo() LocaleInfo { return platform.ActiveInfo() }

func Locales() []string { return platform.Locales() }

func LocaleInfos() []LocaleInfo { return platform.LocaleInfos() }
