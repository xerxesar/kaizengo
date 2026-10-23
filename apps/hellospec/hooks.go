package hellospec

import (
	"fmt"
	"log/slog"
	"strings"

	"kaizengo/internal/engine"
)

const greetingPrefix = "Hello, "

// trimGreetingMessage normalizes whitespace before spec validation runs.
func trimGreetingMessage(hc engine.HookContext) error {
	msg, ok := hc.Fields["message"].(string)
	if !ok {
		return nil
	}
	hc.Fields["message"] = strings.TrimSpace(msg)
	return nil
}

// ensureGreetingPrefix adds a default prefix when the author omits it.
func ensureGreetingPrefix(hc engine.HookContext) error {
	msg, ok := hc.Fields["message"].(string)
	if !ok || msg == "" {
		return nil
	}
	if strings.HasPrefix(strings.ToLower(msg), "hello") {
		return nil
	}
	hc.Fields["message"] = greetingPrefix + msg
	return nil
}

// rejectProfanity blocks messages that fail a custom rule not expressible in app.yaml.
func rejectProfanity(hc engine.HookContext) error {
	msg, _ := hc.Fields["message"].(string)
	if strings.Contains(strings.ToLower(msg), "badword") {
		return fmt.Errorf("message contains blocked content")
	}
	return nil
}

// protectPinnedGreetings reads the projected row and aborts protected deletes.
func protectPinnedGreetings(hc engine.HookContext) error {
	msg, _ := hc.Record["message"].(string)
	if strings.Contains(msg, "[protected]") {
		return fmt.Errorf("protected greetings cannot be deleted")
	}
	return nil
}

// logGreetingCreated runs after projection; hc.Record is the stored row.
func logGreetingCreated(hc engine.HookContext) error {
	slog.Info("hellospec greeting created",
		"id", hc.RecordID,
		"orgId", hc.OrgID,
		"authorId", hc.UserID,
		"message", hc.Record["message"],
	)
	return nil
}

// logGreetingDeleted runs after the soft-delete projection.
func logGreetingDeleted(hc engine.HookContext) error {
	slog.Info("hellospec greeting deleted", "id", hc.RecordID, "orgId", hc.OrgID)
	return nil
}

func beforeCreateGreeting(hc engine.HookContext) error {
	for _, fn := range []func(engine.HookContext) error{
		trimGreetingMessage,
		ensureGreetingPrefix,
		rejectProfanity,
	} {
		if err := fn(hc); err != nil {
			return err
		}
	}
	return nil
}

func beforeUpdateGreeting(hc engine.HookContext) error {
	for _, fn := range []func(engine.HookContext) error{
		trimGreetingMessage,
		rejectProfanity,
	} {
		if err := fn(hc); err != nil {
			return err
		}
	}
	return nil
}

func greetingHooks() engine.Hooks {
	return engine.Hooks{
		BeforeCreate: beforeCreateGreeting,
		AfterCreate:  logGreetingCreated,
		BeforeUpdate: beforeUpdateGreeting,
		BeforeDelete: protectPinnedGreetings,
		AfterDelete:  logGreetingDeleted,
	}
}
