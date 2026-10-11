package github_releases_sdk

import (
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
