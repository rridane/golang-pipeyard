package github_releases_sdk

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

// The chain cannot return an error, so a failure has to survive until Err.
func TestChain_FirstFailureSurvivesToErr(t *testing.T) {
	release := NewGithubReleaseSDK(Config{Owner: "corp", Repo: "svc", Token: "t"}).
		PrepareRelease().
		CreateTag().
		CreateRelease().
		UploadAssets([]string{"dist/app.jar"}).
		Publish()

	err := release.Err()
	if err == nil {
		t.Fatal("a chain started without a tag name must report a failure")
	}
	if !strings.Contains(err.Error(), "tag name is required") {
		t.Fatalf("the first failure must be the one reported, got: %v", err)
	}
}

// PrepareRelease used to panic on a missing tag name.
func TestPrepareRelease_MissingTagNameDoesNotPanic(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("a missing tag name must be an error, not a panic: %v", recovered)
		}
	}()

	if err := NewGithubReleaseSDK(Config{Owner: "corp", Repo: "svc"}).PrepareRelease().Err(); err == nil {
		t.Fatal("a missing tag name must be reported")
	}
}

func TestUploadAssets_RefusesAnUncreatedRelease(t *testing.T) {
	err := NewGithubReleaseSDK(Config{Owner: "corp", Repo: "svc", Token: "t"}).
		WithTag("v1.0.0").
		PrepareRelease().
		UploadAssets([]string{"dist/app.jar"}).
		Err()

	if err == nil || !strings.Contains(err.Error(), "must be created first") {
		t.Fatalf("uploading to an uncreated release must be reported, got: %v", err)
	}
}

func TestSummarise_TruncatesALongMessage(t *testing.T) {
	long := strings.Repeat("x", changelogLineLength+50)

	got := summarise(long)

	if len(got) != changelogLineLength+3 || !strings.HasSuffix(got, "...") {
		t.Fatalf("want a %d-char prefix and an ellipsis, got %d chars", changelogLineLength, len(got))
	}
	if summarise("short") != "short" {
		t.Fatal("a short message must be left alone")
	}
}

// fakeGitHub answers the three calls the chain makes and records every path it
// was asked for, so a test can assert what was not called.
func fakeGitHub(t *testing.T, releases string) (*ReleaseFactory, *[]string) {
	t.Helper()

	var paths []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)

		switch {
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(releases))
		case r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"id": 777, "html_url": "created", "draft": true}`))
		case r.Method == http.MethodPatch:
			_, _ = w.Write([]byte(`{"id": 777, "draft": false}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	return NewGithubReleaseSDK(Config{
		Token: "t", Owner: "acme", Repo: "widgets",
		Enterprise: true, BaseURL: server.URL,
	}), &paths
}

// The chain must never ask for /releases/tags/{tag}: that endpoint answers 404
// for a draft, which is what the release is until the asset is attached.
func TestChain_NeverLooksTheReleaseUpByItsTag(t *testing.T) {
	factory, paths := fakeGitHub(t, `[]`)

	release := factory.WithTag("v1.2.3").PrepareRelease().CreateRelease().Publish()

	if err := release.Err(); err != nil {
		t.Fatalf("the chain must succeed: %v", err)
	}

	for _, path := range *paths {
		if strings.Contains(path, "/releases/tags/") {
			t.Errorf("the chain looked the release up by its tag: %s", path)
		}
	}

	// Published by id, which is the only address a draft has.
	if !slices.Contains(*paths, "PATCH /api/v3/repos/acme/widgets/releases/777") {
		t.Errorf("want the release edited by its id, got %v", *paths)
	}
}

// A replayed tag adopts the draft it already made instead of making a second.
func TestCreateRelease_AdoptsAnExistingDraft(t *testing.T) {
	factory, paths := fakeGitHub(t,
		`[{"id": 409, "tag_name": "v1.2.3", "draft": true, "html_url": "existing"}]`)

	release := factory.WithTag("v1.2.3").PrepareRelease().CreateRelease()

	if err := release.Err(); err != nil {
		t.Fatalf("adopting must succeed: %v", err)
	}

	for _, path := range *paths {
		if strings.HasPrefix(path, "POST ") {
			t.Errorf("a second release was created: %s", path)
		}
	}

	if got := release.(*Release).id; got != 409 {
		t.Errorf("want the existing release's id 409, got %d", got)
	}
}
