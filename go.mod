module smoothie

go 1.16

replace github.com/charmbracelet/charm => ../charm

replace github.com/charmbracelet/bubbletea => ../bubbletea

require (
	github.com/charmbracelet/bubbles v0.8.0
	github.com/charmbracelet/bubbletea v0.14.0
	github.com/charmbracelet/charm v0.8.6
	github.com/charmbracelet/glamour v0.3.0
	github.com/charmbracelet/lipgloss v0.2.1
	github.com/dustin/go-humanize v1.0.0
	github.com/gliderlabs/ssh v0.3.3
	github.com/go-git/go-git/v5 v5.4.2
	github.com/meowgorithm/babyenv v1.3.0
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97
)
