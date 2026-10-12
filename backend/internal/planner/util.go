package planner

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"
)

func NowISO() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func newID(prefix string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	s := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s-%s", prefix, s[0:8], s[8:12], s[12:16], s[16:20], s[20:32])
}

func DefaultDisplayName() string {
	for _, k := range []string{"USERNAME", "USER"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return "You"
}

func normalizeTags(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, raw := range values {
		t := strings.TrimSpace(raw)
		k := strings.ToLower(t)
		if t != "" && !seen[k] {
			seen[k] = true
			out = append(out, t)
		}
	}
	if out == nil {
		out = []string{}
	}
	return out
}

func validateAP(ap *ActionPoint) error {
	ap.Title = strings.TrimSpace(ap.Title)
	if ap.Title == "" {
		return errors.New("AP title is required")
	}
	if len([]rune(ap.Title)) > 200 {
		return errors.New("AP title must be 200 characters or fewer")
	}
	if ap.Importance < 0 || ap.Importance > 100 {
		return errors.New("importance must be between 0 and 100")
	}
	if ap.BaseUrgency < 0 || ap.BaseUrgency > 100 {
		return errors.New("baseUrgency must be between 0 and 100")
	}
	ap.Tags = normalizeTags(ap.Tags)
	return nil
}

func validateDaily(item *DailyCheckpoint) error {
	item.Title = strings.TrimSpace(item.Title)
	if item.Title == "" {
		return errors.New("Daily checkpoint title is required")
	}
	if len([]rune(item.Title)) > 120 {
		return errors.New("Daily checkpoint title must be 120 characters or fewer")
	}
	seen := map[string]bool{}
	clean := make([]string, 0, len(item.CompletedDates))
	for _, d := range item.CompletedDates {
		d = strings.TrimSpace(d)
		if d != "" && !seen[d] {
			seen[d] = true
			clean = append(clean, d)
		}
	}
	item.CompletedDates = clean
	return nil
}

func parseTime(value string) time.Time {
	v := strings.TrimSpace(value)
	if v == "" {
		return time.Unix(1<<62, 0)
	}
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t
		}
		if t, err := time.ParseInLocation(layout, v, time.Local); err == nil {
			return t
		}
	}
	return time.Unix(1<<62, 0)
}

func deadlinePressure(deadline string) float64 {
	if strings.TrimSpace(deadline) == "" {
		return 0
	}
	due := parseTime(deadline)
	if due.Year() >= 9999 {
		return 0
	}
	hours := time.Until(due).Hours()
	if hours <= 0 {
		return 80
	}
	if hours >= 168 {
		return 0
	}
	if hours >= 72 {
		return 15 * ((168 - hours) / 96)
	}
	if hours >= 24 {
		return 15 + 20*((72-hours)/48)
	}
	return 35 + 35*((24-hours)/24)
}

func effectiveUrgency(ap ActionPoint) int {
	v := float64(ap.BaseUrgency) + deadlinePressure(ap.Deadline)
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	return int(math.Round(v))
}

func validateDateKey(v string) error {
	_, err := time.Parse("2006-01-02", strings.TrimSpace(v))
	if err != nil {
		return errors.New("dateKey must be YYYY-MM-DD")
	}
	return nil
}

func cloneState(in PlannerState) PlannerState {
	// Value-only structs plus copied slices/pointers where mutation matters.
	out := in
	out.ActionPoints = cloneAPs(in.ActionPoints)
	out.Archive = cloneAPs(in.Archive)
	out.DailyCheckpoints = make([]DailyCheckpoint, len(in.DailyCheckpoints))
	for i, d := range in.DailyCheckpoints {
		out.DailyCheckpoints[i] = d
		out.DailyCheckpoints[i].CompletedDates = append([]string{}, d.CompletedDates...)
	}
	out.JiraWatch = append([]JiraWatchItem{}, in.JiraWatch...)
	return out
}

func cloneAPs(in []ActionPoint) []ActionPoint {
	out := make([]ActionPoint, len(in))
	for i, ap := range in {
		out[i] = ap
		out[i].Tags = append([]string{}, ap.Tags...)
		if ap.Jira != nil {
			j := *ap.Jira
			out[i].Jira = &j
		}
	}
	return out
}

func sortDatesKeepLast400(values []string) []string {
	if len(values) <= 400 {
		return values
	}
	sort.Strings(values)
	return append([]string{}, values[len(values)-400:]...)
}
