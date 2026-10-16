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

// One hole carries a GO- entry, a CVE- and a GHSA-, and the databases return
// all three. Counting entries would treble every number.
const aliased = `{"results":[
  {"source":{"path":"/src/go.mod"},"packages":[
    {"package":{"name":"golang.org/x/crypto","version":"0.43.0"},
     "vulnerabilities":[
       {"id":"GO-2025-4134","aliases":["CVE-2025-58181","GHSA-j5w8-q4qc-rx2x"]},
       {"id":"GHSA-j5w8-q4qc-rx2x","aliases":["CVE-2025-58181","GO-2025-4134"]},
       {"id":"CVE-2025-58181","aliases":["GO-2025-4134"]},
       {"id":"GO-2025-4135","aliases":["CVE-2025-47914"]}
     ]}
  ]}
]}`

func TestParseAdvisories_CountsOneHoleOnce(t *testing.T) {
	advisories, err := parseAdvisories([]byte(aliased))
	if err != nil {
		t.Fatal(err)
	}

	if len(advisories) != 2 {
		got := make([]string, 0, len(advisories))
		for _, a := range advisories {
			got = append(got, a.ID)
		}
		t.Fatalf("four entries for two holes should give two advisories, got %v", got)
	}
}

// The same identifier against two packages is two advisories: each is fixed on
// its own.
func TestParseAdvisories_KeepsTheSameHoleInTwoPackages(t *testing.T) {
	shared := `{"results":[{"source":{"path":"/src/go.mod"},"packages":[
	  {"package":{"name":"a","version":"1"},"vulnerabilities":[{"id":"GO-1","aliases":["CVE-1"]}]},
	  {"package":{"name":"b","version":"2"},"vulnerabilities":[{"id":"GO-1","aliases":["CVE-1"]}]}
	]}]}`

	advisories, err := parseAdvisories([]byte(shared))
	if err != nil {
		t.Fatal(err)
	}
	if len(advisories) != 2 {
		t.Fatalf("want one per package, got %v", advisories)
	}
}
