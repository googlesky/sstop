package ui

import (
	"fmt"
	"testing"
)

// TestProcessTableLayout verifies that the process table column widths sum to
// the terminal width exactly.
func TestProcessTableLayout(t *testing.T) {
	// fixedW formula from process_table.go render()
	fixedW := colPidW + colGraphW + colUpW + colDownW + colConnsW + colListenW + 6 + 2

	for _, width := range []int{80, 100, 120, 160, 200} {
		nameW := width - fixedW
		if nameW < 10 {
			nameW = 10
		}

		// Data row: indent(2) + PID(8) + gap + NAME(nameW) + gap + GRAPH(16) + gap
		//   + upBar(5) + gap + upText(6) + gap + downBar(5) + gap + downText(6)
		//   + gap + CONNS(6) + gap + LISTEN(6)
		rowW := 2 + colPidW + 1 + nameW + 1 + colGraphW + 1 +
			5 + 1 + 6 + 1 + // up section
			5 + 1 + 6 + 1 + // down section
			colConnsW + 1 + colListenW

		if nameW >= 10 && rowW != width {
			t.Errorf("ProcessTable width=%d: rowW=%d (diff=%d)", width, rowW, rowW-width)
		}
	}
}

// TestRemoteHostsLayout verifies that the remote hosts table column widths
// sum to the terminal width exactly.
func TestRemoteHostsLayout(t *testing.T) {
	// fixedW formula from remote_hosts.go render()
	fixedW := 2 + rhUpW + rhDownW + rhConnsW + rhProcsW + 4

	for _, width := range []int{80, 100, 120, 160, 200} {
		hostW := width - fixedW
		if hostW < 15 {
			hostW = 15
		}

		// Data row: indent(2) + HOST(hostW) + gap + upBar(5) + gap + upText(6)
		//   + gap + downBar(5) + gap + downText(6) + gap + CONNS(6) + gap + PROCS(20)
		rowW := 2 + hostW + 1 +
			5 + 1 + 6 + 1 + // up section
			5 + 1 + 6 + 1 + // down section
			rhConnsW + 1 + rhProcsW

		if hostW >= 15 && rowW != width {
			t.Errorf("RemoteHosts width=%d: rowW=%d (diff=%d)", width, rowW, rowW-width)
		}
	}
}

// TestListenPortsLayout verifies that the listen ports table column widths
// sum to the terminal width exactly.
func TestListenPortsLayout(t *testing.T) {
	// fixedW formula from listen_ports.go render()
	fixedW := lpProtoW + lpPidW + lpProcW + 3 + 2

	for _, width := range []int{80, 100, 120, 160, 200} {
		addrW := width - fixedW
		cmdW := 0
		if addrW > 40 {
			cmdW = addrW / 3
			addrW = addrW - cmdW - 1 // -1 for gap
		}
		if addrW < 15 {
			addrW = 15
		}

		// Data row: indent(2) + PROTO(5) + gap + ADDR(addrW) + gap + PID(8)
		//   + gap + PROCESS(20) [+ gap + CMD(cmdW)]
		rowW := 2 + lpProtoW + 1 + addrW + 1 + lpPidW + 1 + lpProcW
		if cmdW > 0 {
			rowW += 1 + cmdW
		}

		if addrW >= 15 && rowW != width {
			t.Errorf("ListenPorts width=%d: rowW=%d cmdW=%d addrW=%d (diff=%d)",
				width, rowW, cmdW, addrW, rowW-width)
		}
	}
}

// TestProcessDetailLayout verifies that the connection table column widths
// sum to the terminal width exactly.
func TestProcessDetailLayout(t *testing.T) {
	for _, width := range []int{80, 100, 120, 160, 200} {
		lay := computeConnLayout(width)

		// Data row: indicator(2) + proto(5)+space + local(localW)+space
		//   + remote(remoteW)+space + state(10)+space + svc(6)+space
		//   + age(7)+space + up(10)+space + down(10)
		rowW := 2 +
			(lay.protoW + 1) +
			(lay.localW + 1) +
			(lay.remoteW + 1) +
			(lay.stateW + 1) +
			(lay.svcW + 1) +
			(lay.ageW + 1) +
			(lay.upW + 1) +
			lay.downW

		// Only check when remaining >= 30 (normal case)
		remaining := width - (lay.protoW + lay.stateW + lay.svcW + lay.ageW + lay.upW + lay.downW + 7 + 2)
		if remaining >= 30 && rowW != width {
			t.Errorf("ProcessDetail width=%d: rowW=%d localW=%d remoteW=%d (diff=%d)",
				width, rowW, lay.localW, lay.remoteW, rowW-width)
		}

		// Verify localW+remoteW = remaining
		if remaining >= 30 {
			got := lay.localW + lay.remoteW
			if got != remaining {
				t.Errorf("ProcessDetail width=%d: localW(%d)+remoteW(%d)=%d, want remaining=%d",
					width, lay.localW, lay.remoteW, got, remaining)
			}
		}
	}
}

// TestFormatRateCompactAlwaysSixChars ensures the fixed-width invariant holds
// across a wide range of input values including edge cases.
func TestFormatRateCompactAlwaysSixChars(t *testing.T) {
	edgeCases := []float64{
		-1, 0, 0.1, 0.9, 1, 2, 10, 42, 100, 999, 1023,
		1024, 1025, 5120, 10239, 10240, 102400, 1048575, 1048576,
		10485760, 104857600, 1073741823, 1073741824, 10737418240,
		107374182400, 1e15,
	}
	for _, bps := range edgeCases {
		result := FormatRateCompact(bps)
		if len(result) != 6 {
			t.Errorf("FormatRateCompact(%v) = %q (len=%d), want len=6", bps, result, len(result))
		}
	}
}

// TestColumnWidthConstants verifies that the Up/Down column width constants
// correctly account for bar + gap + text.
func TestColumnWidthConstants(t *testing.T) {
	barW := 5
	gapW := 1
	textW := 6 // FormatRateCompact always 6 chars

	tests := []struct {
		name     string
		constVal int
	}{
		{"colUpW", colUpW},
		{"colDownW", colDownW},
		{"rhUpW", rhUpW},
		{"rhDownW", rhDownW},
	}

	expected := barW + gapW + textW
	for _, tt := range tests {
		if tt.constVal != expected {
			t.Errorf("%s = %d, want %d (bar=%d + gap=%d + text=%d)",
				tt.name, tt.constVal, expected, barW, gapW, textW)
		}
	}
}

// TestLayoutConsistencyAcrossWidths ensures layouts work across many widths.
func TestLayoutConsistencyAcrossWidths(t *testing.T) {
	for width := 60; width <= 250; width++ {
		t.Run(fmt.Sprintf("w%d", width), func(t *testing.T) {
			// Process table
			ptFixedW := colPidW + colGraphW + colUpW + colDownW + colConnsW + colListenW + 6 + 2
			nameW := width - ptFixedW
			if nameW >= 10 {
				rowW := 2 + colPidW + 1 + nameW + 1 + colGraphW + 1 + 5 + 1 + 6 + 1 + 5 + 1 + 6 + 1 + colConnsW + 1 + colListenW
				if rowW != width {
					t.Errorf("ProcessTable: rowW=%d != width=%d", rowW, width)
				}
			}

			// Remote hosts
			rhFixedW := 2 + rhUpW + rhDownW + rhConnsW + rhProcsW + 4
			hostW := width - rhFixedW
			if hostW >= 15 {
				rowW := 2 + hostW + 1 + 5 + 1 + 6 + 1 + 5 + 1 + 6 + 1 + rhConnsW + 1 + rhProcsW
				if rowW != width {
					t.Errorf("RemoteHosts: rowW=%d != width=%d", rowW, width)
				}
			}

			// Listen ports
			lpFixedW := lpProtoW + lpPidW + lpProcW + 3 + 2
			addrW := width - lpFixedW
			cmdW := 0
			if addrW > 40 {
				cmdW = addrW / 3
				addrW = addrW - cmdW - 1
			}
			if addrW >= 15 {
				rowW := 2 + lpProtoW + 1 + addrW + 1 + lpPidW + 1 + lpProcW
				if cmdW > 0 {
					rowW += 1 + cmdW
				}
				if rowW != width {
					t.Errorf("ListenPorts: rowW=%d != width=%d (addrW=%d cmdW=%d)", rowW, width, addrW, cmdW)
				}
			}

			// Process detail
			lay := computeConnLayout(width)
			remaining := width - (lay.protoW + lay.stateW + lay.svcW + lay.ageW + lay.upW + lay.downW + 7 + 2)
			if remaining >= 30 {
				rowW := 2 + (lay.protoW + 1) + (lay.localW + 1) + (lay.remoteW + 1) + (lay.stateW + 1) + (lay.svcW + 1) + (lay.ageW + 1) + (lay.upW + 1) + lay.downW
				if rowW != width {
					t.Errorf("ProcessDetail: rowW=%d != width=%d", rowW, width)
				}
			}
		})
	}
}
