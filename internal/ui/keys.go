package ui

import "github.com/charmbracelet/bubbletea"

type keyAction int

const (
	keyNone keyAction = iota
	keyQuit
	keyUp
	keyDown
	keyEnter
	keyEsc
	keySortNext
	keySearch
	keyHelp
	keyPageUp
	keyPageDown
	keyHome
	keyEnd
	keyPause
	keyToggleDNS
	keyNextIface
	keyRemoteHosts
	keyListenPorts
	keyKillProcess
	keyIntervalUp      // faster refresh
	keyIntervalDown    // slower refresh
	keyCumulative      // toggle cumulative mode
	keyTreeToggle      // toggle process tree view
	keySetAlert        // set bandwidth alert
	keySpeedUp         // playback speed up
	keySpeedDown       // playback speed down
	keyGroupView       // docker/systemd group view
)

func matchKey(msg tea.KeyMsg) keyAction {
	switch msg.String() {
	case "q", "ctrl+c":
		return keyQuit
	case "k", "up":
		return keyUp
	case "j", "down":
		return keyDown
	case "enter":
		return keyEnter
	case "esc":
		return keyEsc
	case "s":
		return keySortNext
	case "/":
		return keySearch
	case "?":
		return keyHelp
	case "pgup", "ctrl+u":
		return keyPageUp
	case "pgdown", "ctrl+d":
		return keyPageDown
	case "home", "g":
		return keyHome
	case "end", "G":
		return keyEnd
	case " ":
		return keyPause
	case "d":
		return keyToggleDNS
	case "i", "tab":
		return keyNextIface
	case "h":
		return keyRemoteHosts
	case "l":
		return keyListenPorts
	case "K":
		return keyKillProcess
	case "+", "=":
		return keyIntervalUp
	case "-":
		return keyIntervalDown
	case "c":
		return keyCumulative
	case "t":
		return keyTreeToggle
	case "A":
		return keySetAlert
	case "right":
		return keySpeedUp
	case "left":
		return keySpeedDown
	case "D":
		return keyGroupView
	}
	return keyNone
}
