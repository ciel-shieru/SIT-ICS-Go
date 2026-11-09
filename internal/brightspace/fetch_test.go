package brightspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type fixtureFetcher struct {
	responses map[string]string
	failures  map[string]error
}

func (f *fixtureFetcher) DecodeJSON(url string, value interface{}) error {
	return f.decode(url, value)
}

func (f *fixtureFetcher) DecodeJSONFast(url string, value interface{}) error {
	return f.decode(url, value)
}

func (f *fixtureFetcher) decode(url string, value interface{}) error {
	for fragment, err := range f.failures {
		if strings.Contains(url, fragment) {
			return err
		}
	}
	for fragment, body := range f.responses {
		if strings.Contains(url, fragment) {
			if err := json.Unmarshal([]byte(body), value); err != nil {
				return fmt.Errorf("decode fixture for %s: %w", fragment, err)
			}
			return nil
		}
	}
	return fmt.Errorf("unexpected fixture URL: %s", url)
}

func TestFetchReturnsPartialDataAndErrorWhenCourseEndpointFails(t *testing.T) {
	fetcher := &fixtureFetcher{
		responses: map[string]string{
			"/d2l/api/le/versions/": `{"LatestVersion":"1.70"}`,
			"/mycourses": `{"Courses":[{"OrgUnitId":"100","Name":"Fake Course One","Code":"ALT2501"},{"OrgUnitId":"200","Name":"Fake Course Two","Code":"ALT2502"}]}`,
			"/100/calendar/events/": `[{"CalendarEventId":1,"OrgUnitId":100,"Title":"Fake Event","StartDateTime":"2026-11-12T10:00:00Z","EndDateTime":"2026-11-12T11:00:00Z"}]`,
			"/100/dropbox/folders/": `[]`,
			"/200/dropbox/folders/": `[{"Id":22,"Name":"Fake Assignment","DueDate":"2026-11-20T23:59:00Z"}]`,
		},
		failures: map[string]error{
			"/200/calendar/events/": errors.New("temporary endpoint failure"),
		},
	}

	events, folders, err := Fetch(context.Background(), NewClient("https://example.test", fetcher))
	if err == nil {
		t.Fatal("Fetch() error = nil, want partial-fetch error")
	}
	if len(events) != 1 || events[0].Title != "Fake Event" || events[0].OrgUnitName != "Fake Course One" {
		t.Fatalf("partial events = %#v, want the successfully fetched course event", events)
	}
	if len(folders) != 1 || folders[0].Name != "Fake Assignment" || folders[0].OrgUnitName != "Fake Course Two" {
		t.Fatalf("folders = %#v, want Dropbox results despite the calendar endpoint failure", folders)
	}
}

func TestFetchQuizzesReturnsPartialDataAndErrorWhenCourseEndpointFails(t *testing.T) {
	fetcher := &fixtureFetcher{
		responses: map[string]string{
			"/d2l/api/le/versions/": `{"LatestVersion":"1.70"}`,
			"/mycourses": `{"Courses":[{"OrgUnitId":"100","Name":"Fake Course One","Code":"ALT2501"},{"OrgUnitId":"200","Name":"Fake Course Two","Code":"ALT2502"}]}`,
			"/100/quizzes/": `{"Objects":[{"QuizId":7,"Name":"Fake Quiz"}]}`,
		},
		failures: map[string]error{
			"/200/quizzes/": errors.New("temporary endpoint failure"),
		},
	}

	quizzes, err := FetchQuizzes(context.Background(), NewClient("https://example.test", fetcher))
	if err == nil {
		t.Fatal("FetchQuizzes() error = nil, want partial-fetch error")
	}
	if len(quizzes) != 1 || quizzes[0].QuizId != 7 || quizzes[0].OrgUnitName != "Fake Course One" {
		t.Fatalf("partial quizzes = %#v, want the successfully fetched course quiz", quizzes)
	}
}
