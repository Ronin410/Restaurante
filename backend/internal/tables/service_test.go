package tables

import (
	"testing"
	"time"
)

func TestWithOccupancyComputesMinutesForOccupiedTable(t *testing.T) {
	openedAt := time.Now().Add(-15 * time.Minute).Format("2006-01-02 15:04:05")
	tbl := Table{Status: "ocupada", OpenedAt: &openedAt}

	withOccupancy(&tbl)

	if tbl.OccupiedMinutes == nil {
		t.Fatal("esperaba OccupiedMinutes calculado para una mesa ocupada")
	}
	if *tbl.OccupiedMinutes < 14 || *tbl.OccupiedMinutes > 16 {
		t.Fatalf("esperaba ~15 minutos, obtuve %d", *tbl.OccupiedMinutes)
	}
}

func TestWithOccupancySkipsFreeTable(t *testing.T) {
	tbl := Table{Status: "libre"}
	withOccupancy(&tbl)
	if tbl.OccupiedMinutes != nil {
		t.Fatal("una mesa libre no debería tener tiempo de ocupación")
	}
}
