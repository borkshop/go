package caskdiskstore

import (
	"borkshop/cask/casktest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

func TestStress(t *testing.T) {
	fs := osfs.New(".cask")
	store := &Store{Filesystem: fs}
	report := casktest.StressStoreConfig{
		Concurrency: 100,
		Duration:    200 * time.Millisecond,
	}.Stress(store)

	t.Logf("%d cycles\nn", report.Cycles)
	t.Logf("%d write errors\n", report.WriteErrors)
	t.Logf("%d read errors\n", report.ReadErrors)
	t.Logf("%d data integrity errors\n", report.DataErrors)

	assert.Equal(t, 0, report.WriteErrors, "write errors")
	assert.Equal(t, 0, report.ReadErrors, "read errors")
	assert.Equal(t, 0, report.DataErrors, "data integrity errors")
	assert.NotEqual(t, 0, report.Cycles, "no cycles")
}
