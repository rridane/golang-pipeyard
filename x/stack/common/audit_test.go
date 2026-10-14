package common

import "testing"

// The report nests a lockfile, the packages it locks and the vulnerabilities
// against each, and a package with none is the common case.
const report = `{"results":[
  {"source":{"path":"/src/dags/uv.lock"},"packages":[
    {"package":{"name":"aiohttp","version":"3.13.5"},
     "vulnerabilities":[{"id":"PYSEC-2026-2104"},{"id":"PYSEC-2026-2105"}]},
    {"package":{"name":"pandas","version":"2.2.0"},"vulnerabilities":[]}
  ]},
  {"source":{"path":"/src/uv.lock"},"packages":[
    {"package":{"name":"torch","version":"2.9.0"},"vulnerabilities":[]}
  ]}
]}`

func TestParseAdvisories(t *testing.T) {
	advisories, err := parseAdvisories([]byte(report))
	if err != nil {
		t.Fatal(err)
	}

	if len(advisories) != 2 {
		t.Fatalf("want two advisories, got %v", advisories)
	}

	// The path has to come back the way the repository writes it, not the way
	// the container saw it.
	if got := advisories[0].Lockfile; got != "dags/uv.lock" {
		t.Errorf("the container root should be trimmed, got %q", got)
	}
	if got, want := advisories[0].String(), "aiohttp 3.13.5: PYSEC-2026-2104 (dags/uv.lock)"; got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// A clean repository reports an empty result, which has to read as no
// advisory rather than as a scanner that never ran.
func TestParseAdvisories_EmptyReportIsClean(t *testing.T) {
	advisories, err := parseAdvisories([]byte(`{"results":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(advisories) != 0 {
		t.Fatalf("want none, got %v", advisories)
	}
}

// Anything that is not a report means the scanner did not run, and that must
// not be mistaken for a clean repository.
func TestParseAdvisories_RefusesWhatIsNotAReport(t *testing.T) {
	for _, output := range []string{"", "osv-scanner: not found", "cat: can't open /tmp/report.json"} {
		if _, err := parseAdvisories([]byte(output)); err == nil {
			t.Errorf("%q must not read as a clean repository", output)
		}
	}
}
