package uci

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Skiddle-Labs/axon-engine/internal/search"
)

// TestUCI_NewOptionsPresence verifies that the newly added UCI options are advertised.
func TestUCI_NewOptionsPresence(t *testing.T) {
	input := "uci\nquit\n"
	var output bytes.Buffer
	u := NewUCI(strings.NewReader(input), &output)
	u.Start()

	got := output.String()

	expectedOptions := []string{
		"option name UCI_ShowWDL type check default false",
		"option name LMR_Base type spin default 75 min 0 max 200",
		"option name LMR_Multiplier type spin default 225 min 0 max 500",
		"option name MC_R type spin default 3 min 1 max 10",
		"option name MC_M type spin default 6 min 1 max 20",
		"option name MC_C type spin default 3 min 1 max 10",
	}

	for _, opt := range expectedOptions {
		if !strings.Contains(got, opt) {
			t.Errorf("Expected UCI option not found in output: %s", opt)
		}
	}
}

// TestUCI_SetNewOptions verifies that setting the new options correctly updates the engine state.
func TestUCI_SetNewOptions(t *testing.T) {
	u := NewUCI(strings.NewReader(""), &bytes.Buffer{})

	// 1. Test UCI_ShowWDL
	u.handleSetOption([]string{"setoption", "name", "UCI_ShowWDL", "value", "true"})
	if !u.showWDL {
		t.Error("Expected showWDL to be true after setoption")
	}

	u.handleSetOption([]string{"setoption", "name", "UCI_ShowWDL", "value", "false"})
	if u.showWDL {
		t.Error("Expected showWDL to be false after setoption")
	}

	// 2. Test LMR_Base (Stored as float64 in search, integer in UCI * 100)
	u.handleSetOption([]string{"setoption", "name", "LMR_Base", "value", "85"})
	if search.LMR_Base != 0.85 {
		t.Errorf("Expected search.LMR_Base to be 0.85, got %f", search.LMR_Base)
	}

	// 3. Test LMR_Multiplier
	u.handleSetOption([]string{"setoption", "name", "LMR_Multiplier", "value", "250"})
	if search.LMR_Multiplier != 2.50 {
		t.Errorf("Expected search.LMR_Multiplier to be 2.50, got %f", search.LMR_Multiplier)
	}

	// 4. Test MC parameters
	u.handleSetOption([]string{"setoption", "name", "MC_R", "value", "4"})
	if search.MC_R != 4 {
		t.Errorf("Expected search.MC_R to be 4, got %d", search.MC_R)
	}

	u.handleSetOption([]string{"setoption", "name", "MC_M", "value", "8"})
	if search.MC_M != 8 {
		t.Errorf("Expected search.MC_M to be 8, got %d", search.MC_M)
	}

	u.handleSetOption([]string{"setoption", "name", "MC_C", "value", "5"})
	if search.MC_C != 5 {
		t.Errorf("Expected search.MC_C to be 5, got %d", search.MC_C)
	}
}

// TestUCI_LMRParameterSync verifies that LMR table is recomputed when parameters change.
func TestUCI_LMRParameterSync(t *testing.T) {
	// Reset to defaults first
	search.ResetSearchParameters()
	initialBase := search.LMR_Base

	u := NewUCI(strings.NewReader(""), &bytes.Buffer{})
	u.handleSetOption([]string{"setoption", "name", "LMR_Base", "value", "100"})

	if search.LMR_Base == initialBase {
		t.Error("search.LMR_Base did not change after setoption")
	}

	if search.LMR_Base != 1.0 {
		t.Errorf("Expected LMR_Base 1.0, got %f", search.LMR_Base)
	}
}
