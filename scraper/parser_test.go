package scraper

import (
	"strings"
	"testing"
	"time"
)

const sampleHTML = `<div class="view-content">
  <div class="views-row">
    <article about="/events/2026/05/13/test-event" class="event event--teaser event--adult teaser">
      <div class="event__details">
        <div class="event__date">
          <div class="field field--name-field-event-date-and-time field__item">
            <span class="date-display-range">Wednesday, 5/13/2026, 10:00 - 11:00</span>
          </div>
        </div>
        <div class="event__name">
          <h2 class="event__title">
            <a href="/events/2026/05/13/test-event" rel="bookmark"><span>Test Event</span>
</a>
          </h2>
        </div>
        <div class="event__location">
          <a href="/locations/mission-bay">Mission Bay</a>
        </div>
      </div>
    </article>
  </div>
</div>`

func TestParseHTML_extractsFields(t *testing.T) {
	instances, err := parseHTML(sampleHTML)
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(instances))
	}
	inst := instances[0]
	if inst.Name != "Test Event" {
		t.Errorf("Name = %q, want %q", inst.Name, "Test Event")
	}
	if inst.URL != "https://sfpl.org/events/2026/05/13/test-event" {
		t.Errorf("URL = %q", inst.URL)
	}
	if inst.Location != "Mission Bay" {
		t.Errorf("Location = %q, want %q", inst.Location, "Mission Bay")
	}
	if inst.Date.Month() != time.May || inst.Date.Day() != 13 || inst.Date.Hour() != 10 {
		t.Errorf("Date = %v", inst.Date)
	}
	if inst.EndTime.Hour() != 11 {
		t.Errorf("EndTime = %v", inst.EndTime)
	}
}

func TestParseDateRange_parsesStartAndEnd(t *testing.T) {
	start, end, err := parseDateRange("Wednesday, 5/13/2026, 10:00 - 11:30")
	if err != nil {
		t.Fatal(err)
	}
	if start.Hour() != 10 || start.Minute() != 0 {
		t.Errorf("start = %v", start)
	}
	if end.Hour() != 11 || end.Minute() != 30 {
		t.Errorf("end = %v", end)
	}
}

func TestParseDateRange_afternoonEventsCorrectlyAsPM(t *testing.T) {
	tests := []struct {
		input      string
		wantStartH int
		wantEndH   int
	}{
		{"Saturday, 5/30/2026, 1:00 - 2:00", 13, 14},   // 1 PM – 2 PM
		{"Sunday, 5/31/2026, 2:00 - 3:30", 14, 15},      // 2 PM – 3:30 PM
		{"Thursday, 6/4/2026, 4:00 - 6:00", 16, 18},     // 4 PM – 6 PM
		{"Thursday, 6/11/2026, 5:30 - 6:30", 17, 18},    // 5:30 PM – 6:30 PM
		{"Saturday, 6/27/2026, 12:00 - 1:00", 12, 13},   // noon – 1 PM
	}
	for _, tt := range tests {
		start, end, err := parseDateRange(tt.input)
		if err != nil {
			t.Errorf("parseDateRange(%q) error: %v", tt.input, err)
			continue
		}
		if start.Hour() != tt.wantStartH {
			t.Errorf("parseDateRange(%q) start hour = %d, want %d", tt.input, start.Hour(), tt.wantStartH)
		}
		if end.Hour() != tt.wantEndH {
			t.Errorf("parseDateRange(%q) end hour = %d, want %d", tt.input, end.Hour(), tt.wantEndH)
		}
	}
}

func TestParseDescription_extractsBodyText(t *testing.T) {
	html := `<html><body>
		<div class="event__content clearfix"><p>Join us for chess!</p></div>
	</body></html>`
	got := ParseDescription(strings.NewReader(html))
	if !strings.Contains(got, "Join us for chess!") {
		t.Errorf("ParseDescription = %q", got)
	}
}
