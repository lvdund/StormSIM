package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGnbMapping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[int]string
		wantErr  bool
	}{
		{
			name:  "valid mapping",
			input: "37:000008, 42:000009,59:000010",
			expected: map[int]string{
				37: "000008",
				42: "000009",
				59: "000010",
			},
			wantErr: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: map[int]string{},
			wantErr:  false,
		},
		{
			name:     "invalid format - missing colon",
			input:    "37-000008",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid format - non-numeric CSV ID",
			input:    "abc:000008",
			expected: nil,
			wantErr:  true,
		},
		{
			name:  "single mapping",
			input: "42:000008",
			expected: map[int]string{
				42: "000008",
			},
			wantErr: false,
		},
		{
			name:  "mapping with spaces",
			input: " 37 : 000008 , 42 : 000009 ",
			expected: map[int]string{
				37: "000008",
				42: "000009",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseGnbMapping(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGnbMapping() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(result) != len(tt.expected) {
					t.Errorf("ParseGnbMapping() len = %d, want %d", len(result), len(tt.expected))
					return
				}
				for k, v := range tt.expected {
					if result[k] != v {
						t.Errorf("ParseGnbMapping() result[%d] = %s, want %s", k, result[k], v)
					}
				}
			}
		})
	}
}

func TestLoadHandoverEventsFromCSV(t *testing.T) {
	// Create a temporary CSV file for testing
	tmpDir := t.TempDir()
	testCSV := filepath.Join(tmpDir, "test_handover.csv")

	csvContent := `Bước,ue0_x,ue0_y,ue0_huong,ue0_BS_ketnoi,ue0_prx_hientai,ue0_speed,ue0_speed_eff,ue0_handover,ue0_handover_to_type
0,7012.3,6472.6,1,42,-52.4,44.46,44.46,0,0
1,7012.3,6472.6,1,42,-52.4,44.46,44.46,0,0
2,7012.3,6472.6,1,37,-52.2,44.46,44.46,1,1
3,7012.3,6472.6,1,37,-52.2,44.46,44.46,0,0
4,7012.3,6472.6,1,59,-52.2,44.46,44.46,1,2
`
	err := os.WriteFile(testCSV, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test CSV: %v", err)
	}

	gnbMapping := map[int]string{
		37: "000008",
		42: "000009",
		59: "000010",
	}

	cfg := CSVHandoverConfig{
		FilePath:     testCSV,
		GnbIdMapping: gnbMapping,
		StepDelayMs:  1000,
	}

	steps, err := LoadHandoverEventsFromCSV(cfg)
	if err != nil {
		t.Fatalf("LoadHandoverEventsFromCSV() error = %v", err)
	}

	// Should have 2 handover events (at steps 2 and 4)
	if len(steps) != 2 {
		t.Errorf("LoadHandoverEventsFromCSV() got %d steps, want 2", len(steps))
	}

	// Verify first handover (step 2: from 42 to 37, Xn type)
	if len(steps) > 0 {
		if steps[0].Step != 2 {
			t.Errorf("First handover step = %d, want 2", steps[0].Step)
		}
		if steps[0].ToGnbId != "000008" { // 37 maps to 000008
			t.Errorf("First handover ToGnbId = %s, want 000008", steps[0].ToGnbId)
		}
		if steps[0].HandoverType != 1 { // Xn
			t.Errorf("First handover type = %d, want 1 (Xn)", steps[0].HandoverType)
		}
	}

	// Verify second handover (step 4: from 37 to 59, N2 type)
	if len(steps) > 1 {
		if steps[1].Step != 4 {
			t.Errorf("Second handover step = %d, want 4", steps[1].Step)
		}
		if steps[1].ToGnbId != "000010" { // 59 maps to 000010
			t.Errorf("Second handover ToGnbId = %s, want 000010", steps[1].ToGnbId)
		}
		if steps[1].HandoverType != 2 { // N2
			t.Errorf("Second handover type = %d, want 2 (N2)", steps[1].HandoverType)
		}
	}
}

func TestLoadHandoverEventsFromCSV_MissingColumns(t *testing.T) {
	tmpDir := t.TempDir()
	testCSV := filepath.Join(tmpDir, "test_missing_cols.csv")

	// CSV missing ue0_handover column
	csvContent := `Bước,ue0_x,ue0_BS_ketnoi
0,7012.3,42
`
	err := os.WriteFile(testCSV, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test CSV: %v", err)
	}

	cfg := CSVHandoverConfig{
		FilePath:     testCSV,
		GnbIdMapping: map[int]string{},
	}

	_, err = LoadHandoverEventsFromCSV(cfg)
	if err == nil {
		t.Error("LoadHandoverEventsFromCSV() should return error for missing columns")
	}
}

func TestHandoverStep_Types(t *testing.T) {
	xnStep := HandoverStep{HandoverType: 1}
	n2Step := HandoverStep{HandoverType: 2}

	if !xnStep.IsXnHandover() {
		t.Error("HandoverStep with type 1 should be Xn handover")
	}
	if xnStep.IsN2Handover() {
		t.Error("HandoverStep with type 1 should not be N2 handover")
	}

	if n2Step.IsXnHandover() {
		t.Error("HandoverStep with type 2 should not be Xn handover")
	}
	if !n2Step.IsN2Handover() {
		t.Error("HandoverStep with type 2 should be N2 handover")
	}
}
