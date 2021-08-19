module smoothie

go 1.16

replace github.com/charmbracelet/charm => ../charm

replace github.com/charmbracelet/bubbletea => ../bubbletea

replace github.com/charmbracelet/wish => ../wish

require (
	github.com/charmbracelet/bubbles v0.8.0
	github.com/charmbracelet/bubbletea v0.14.1
	github.com/charmbracelet/glamour v0.3.0
	github.com/charmbracelet/lipgloss v0.3.1-0.20210819193614-7f051d0e92a3
	github.com/charmbracelet/wish v0.0.0-20210816211645-088e4d8b1b04
	github.com/dustin/go-humanize v1.0.0
	github.com/gliderlabs/ssh v0.3.3
	github.com/go-git/go-git/v5 v5.4.2
	github.com/meowgorithm/babyenv v1.3.0
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.9.0
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97 // indirect
)
