package journalhook

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/coreos/go-systemd/journal"
	"github.com/sirupsen/logrus"
)

type journalHook struct {
	levels    []logrus.Level
	formatter logrus.Formatter
}

var (
	// We are logging to file, strip colors to make the output more readable
	txtFormatter = &logrus.TextFormatter{DisableColors: true}

	severityMap = map[logrus.Level]journal.Priority{
		logrus.DebugLevel: journal.PriDebug,
		logrus.InfoLevel:  journal.PriInfo,
		logrus.WarnLevel:  journal.PriWarning,
		logrus.ErrorLevel: journal.PriErr,
		logrus.FatalLevel: journal.PriCrit,
		logrus.PanicLevel: journal.PriEmerg,
	}

	allLevels = []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
)

func stringifyOp(r rune) rune {
	// Journal wants uppercase strings. See `validVarName`
	// https://github.com/coreos/go-systemd/blob/ff118ad0f8d9cf99903d3391ca3a295671022cee/journal/journal.go#L137-L147
	switch {
	case r >= 'A' && r <= 'Z':
		return r
	case r >= '0' && r <= '9':
		return r
	case r == '_':
		return r
	case r >= 'a' && r <= 'z':
		return r - 32
	default:
		return rune('_')
	}
}

func stringifyKey(key string) string {
	key = strings.Map(stringifyOp, key)
	if strings.HasPrefix(key, "_") {
		key = strings.TrimPrefix(key, "_")
	}
	return key
}

// Journal wants strings but logrus takes anything.
func stringifyEntries(data map[string]interface{}) map[string]string {
	entries := make(map[string]string)
	for k, v := range data {

		key := stringifyKey(k)
		entries[key] = fmt.Sprint(v)
	}
	return entries
}

func (hook *journalHook) Fire(entry *logrus.Entry) error {
	return journal.Send(entry.Message, severityMap[entry.Level], stringifyEntries(entry.Data))
}

// `Levels()` returns a slice of `Levels` the hook is fired for.
func (hook *journalHook) Levels() []logrus.Level {
	if len(hook.levels) == 0 {
		return allLevels
	}

	return hook.levels
}

func New(levels []logrus.Level) *journalHook {
	return &journalHook{
		levels:    levels,
		formatter: txtFormatter,
	}
}

// Adds the Journal hook if journal is enabled
// Sets log output to ioutil.Discard so stdout isn't captured.
func Enable() {
	if !journal.Enabled() {
		logrus.Warning("Journal not available but user requests we log to it. Ignoring")
	} else {
		logrus.AddHook(New(allLevels))
		logrus.SetOutput(ioutil.Discard)
	}
}
