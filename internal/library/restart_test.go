package library

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSaveCommittedOriginalAndRecoverySurviveAbruptExit(t *testing.T) {
	if os.Getenv("SKILLVIEW_SAVE_CRASH_HELPER") == "1" {
		l, err := New(os.Getenv("SKILLVIEW_SAVE_CRASH_DATA"), nil)
		if err == nil {
			var catalog Catalog
			catalog, err = l.Scan(context.Background())
			if err == nil && len(catalog.Skills) == 1 {
				var doc Document
				doc, err = l.Read(catalog.Skills[0].ID, "SKILL.md")
				if err == nil {
					_, err = l.Save(doc.SkillID, doc.Path, doc.Revision, fixture+"Saved before crash.\n")
				}
			} else if err == nil {
				err = fmt.Errorf("unexpected fixture catalog")
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(24)
		}
		// Simulate process loss after disk commit, without application cleanup.
		os.Exit(23)
	}
	l, original, _, id := setup(t)
	baseline, err := l.Read(id, "SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(executable, "-test.run=^TestSaveCommittedOriginalAndRecoverySurviveAbruptExit$")
	command.Env = append(os.Environ(), "SKILLVIEW_SAVE_CRASH_HELPER=1", "SKILLVIEW_SAVE_CRASH_DATA="+l.dataDir)
	output, err := command.CombinedOutput()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 23 {
		t.Fatal("fixture did not exit after commit", string(output), err)
	}
	data, err := os.ReadFile(filepath.Join(original, "SKILL.md"))
	if err != nil || string(data) != fixture+"Saved before crash.\n" {
		t.Fatal("committed text lost", string(data), err)
	}
	restarted, err := New(l.dataDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := restarted.Scan(context.Background())
	if err != nil || len(catalog.Skills) != 1 || catalog.Skills[0].Revision == baseline.Revision {
		t.Fatal("restart did not validate current disk", catalog, err)
	}
	result, err := restarted.Save(catalog.Skills[0].ID, "SKILL.md", baseline.Revision, "stale draft")
	if err != nil || result.Status != "conflict" {
		t.Fatal("stale draft replaced committed text", result, err)
	}
	recovery, err := filepath.Glob(filepath.Join(l.dataDir, "Recovery", "original-*.txt"))
	if err != nil || len(recovery) != 1 {
		t.Fatal("recovery copy lost", recovery, err)
	}
	previous, err := os.ReadFile(recovery[0])
	if err != nil || string(previous) != fixture {
		t.Fatal("recovery bytes lost", string(previous), err)
	}
}
