package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	cursorStyle = focusedStyle.Copy()
	noStyle     = lipgloss.NewStyle()
	helpStyle   = blurredStyle.Copy()

	focusedButton = focusedStyle.Copy().Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
)

var (
	httpreq *HttpRequest
	token   string
)

func init() {
	httpreq = NewHttpRequest()
}

type state int

type user struct {
	ID    int
	token string
}

type LoginModel struct {
	focusIndex    int
	usernameInput textinput.Model
	passwordInput textinput.Model
	submitButton  string
	isLoading     bool
	spinner       spinner.Model
	sub           chan loginMsg
	msg           string
	cursorMode    textinput.CursorMode
}

func MakeLoginModel() LoginModel {
	m := LoginModel{
		focusIndex:    int(usernameLoginState),
		usernameInput: textinput.New(),
		passwordInput: textinput.New(),
		submitButton:  blurredButton,
		isLoading:     false,
		spinner:       spinner.New(),
		sub:           make(chan loginMsg),
	}

	m.usernameInput.Placeholder = "username"
	m.usernameInput.Focus()
	m.usernameInput.PromptStyle = focusedStyle
	m.usernameInput.TextStyle = focusedStyle
	m.usernameInput.CursorStyle = cursorStyle

	m.passwordInput.Placeholder = "password"
	m.passwordInput.EchoMode = textinput.EchoPassword
	m.passwordInput.EchoCharacter = '*'
	m.passwordInput.CursorStyle = cursorStyle

	m.spinner.Spinner = spinner.MiniDot
	return m
}

func (m LoginModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m LoginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case loginMsg:
		m.isLoading = false
		m.spinner.Finish()
		if msg.err != nil {
			m.msg = msg.err.Error()
		} else {
			m.msg = "logged in"
			httpreq.token = msg.msg
		}

		cmds = make([]tea.Cmd, 0, 1)
		cmds = append(cmds, m.spinner.Tick)

		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		s := msg.String()
		switch s {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit

		case "up", "down", "enter", "tab", "shift+tab":
			if s == "enter" && m.focusIndex == int(submitLoginState) {
				m.isLoading = true
				m.msg = ""
				cmds = make([]tea.Cmd, 0, 2)
				cmds = append(cmds, m.login())
				cmds = append(cmds, m.spinner.Tick)
				return m, tea.Batch(cmds...)
			}

			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > 2 {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = 2
			}

			m.passwordInput.PromptStyle = noStyle
			m.passwordInput.TextStyle = noStyle

			m.usernameInput.PromptStyle = noStyle
			m.usernameInput.TextStyle = noStyle
			m.passwordInput.Blur()
			m.usernameInput.Blur()

			cmds = make([]tea.Cmd, 0, 2)
			if m.focusIndex == int(usernameLoginState) {
				m.usernameInput.PromptStyle = focusedStyle
				m.usernameInput.TextStyle = focusedStyle

				m.passwordInput.Blur()
				cmds = append(cmds, m.usernameInput.Focus())

			} else if m.focusIndex == int(passwordLoginState) {
				m.passwordInput.PromptStyle = focusedStyle
				m.passwordInput.TextStyle = focusedStyle

				m.usernameInput.Blur()
				cmds = append(cmds, m.passwordInput.Focus())
			}
			return m, tea.Batch(cmds...)
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	}

	cmds = make([]tea.Cmd, 2)

	m.usernameInput, cmds[0] = m.usernameInput.Update(msg)
	m.passwordInput, cmds[1] = m.passwordInput.Update(msg)

	return m, tea.Batch(cmds...)
}

func (m LoginModel) View() string {
	var b strings.Builder
	b.WriteString(m.usernameInput.View())
	b.WriteRune('\n')
	b.WriteString(m.passwordInput.View())
	b.WriteRune('\n')

	button := &blurredButton
	if m.focusIndex == 2 {
		button = &focusedButton
	}

	fmt.Fprintf(&b, "\n%s\n", *button)

	if m.isLoading {
		b.WriteString(fmt.Sprintf("%s Loading...\n", m.spinner.View()))
	} else {
		b.WriteRune('\n')
	}

	b.WriteString(fmt.Sprintf("%s", m.msg))
	return b.String()
}

func (m *LoginModel) login() tea.Cmd {
	return func() tea.Msg {
		username := m.usernameInput.Value()
		password := m.passwordInput.Value()
		token, err := httpreq.LoginScele(username, password)

		if err != nil {
			return loginMsg{
				err: err,
			}
		}

		return loginMsg{
			msg: token,
			err: nil,
		}

	}
}

type ForumModel struct {
}

//TODO: bikin submodel buat login, forum dll
type model struct {
	state      state
	user       user
	forumModel ForumModel
	loginModel LoginModel
	token      string
}

const (
	usernameLoginState state = iota
	passwordLoginState
	submitLoginState
)

const (
	loginState state = iota
	forumState
)

type loginMsg struct {
	msg string
	err error
}

func initialModel() model {
	m := model{
		state:      loginState,
		loginModel: MakeLoginModel(),
	}

	return m
}

func (m model) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	cmds = append(cmds, m.loginModel.Init())
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case loginState:
		return m.loginModel.Update(msg)
	}
	return nil, nil
}

func (m model) View() string {
	switch m.state {
	case loginState:
		return m.loginModel.View()
	}
	return ""
}

func main() {
	if err := tea.NewProgram(initialModel()).Start(); err != nil {
		fmt.Printf("could not start program: %s\n", err)
		os.Exit(1)
	}
}
