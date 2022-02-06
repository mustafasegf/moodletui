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

func login(m model) tea.Cmd {
	return func() tea.Msg {
		username := m.loginModel.usernameInput.Value()
		password := m.loginModel.passwordInput.Value()
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

func initialModel() model {
	m := model{
		state: loginState,
		loginModel: LoginModel{
			focusIndex:    int(usernameLoginState),
			usernameInput: textinput.New(),
			passwordInput: textinput.New(),
			submitButton:  blurredButton,
			isLoading:     false,
			spinner:       spinner.New(),
			sub:           make(chan loginMsg),
		},
	}

	m.loginModel.usernameInput.Placeholder = "username"
	m.loginModel.usernameInput.Focus()
	m.loginModel.usernameInput.PromptStyle = focusedStyle
	m.loginModel.usernameInput.TextStyle = focusedStyle

	m.loginModel.passwordInput.Placeholder = "password"
	m.loginModel.passwordInput.EchoMode = textinput.EchoPassword
	m.loginModel.passwordInput.EchoCharacter = '*'

	m.loginModel.spinner.Spinner = spinner.MiniDot
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case loginMsg:
		m.loginModel.isLoading = false
		m.loginModel.spinner.Finish()
		if msg.err != nil {
			m.loginModel.msg = msg.err.Error()
		} else {
			m.loginModel.msg = "logged in"
			m.token = msg.msg
		}

		cmds = make([]tea.Cmd, 0, 1)
		cmds = append(cmds, m.loginModel.spinner.Tick)

		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		s := msg.String()
		switch s {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit

		case "up", "down", "enter", "tab", "shift+tab":
			if s == "enter" && m.loginModel.focusIndex == int(submitLoginState) {
				m.loginModel.isLoading = true
				m.loginModel.msg = ""
				cmds = make([]tea.Cmd, 0, 2)
				cmds = append(cmds, login(m))
				cmds = append(cmds, m.loginModel.spinner.Tick)
				return m, tea.Batch(cmds...)
			}

			if s == "up" || s == "shift+tab" {
				m.loginModel.focusIndex--
			} else {
				m.loginModel.focusIndex++
			}

			if m.loginModel.focusIndex > 2 {
				m.loginModel.focusIndex = 0
			} else if m.loginModel.focusIndex < 0 {
				m.loginModel.focusIndex = 2
			}

			m.loginModel.passwordInput.PromptStyle = noStyle
			m.loginModel.passwordInput.TextStyle = noStyle

			m.loginModel.usernameInput.PromptStyle = noStyle
			m.loginModel.usernameInput.TextStyle = noStyle
			m.loginModel.passwordInput.Blur()
			m.loginModel.usernameInput.Blur()

			cmds = make([]tea.Cmd, 0, 2)
			if m.loginModel.focusIndex == int(usernameLoginState) {
				m.loginModel.usernameInput.PromptStyle = focusedStyle
				m.loginModel.usernameInput.TextStyle = focusedStyle

				cmds = append(cmds, m.loginModel.usernameInput.Focus())
			} else if m.loginModel.focusIndex == int(passwordLoginState) {

				m.loginModel.passwordInput.PromptStyle = focusedStyle
				m.loginModel.passwordInput.TextStyle = focusedStyle

				cmds = append(cmds, m.loginModel.passwordInput.Focus())
			}
		default:

			cmds = make([]tea.Cmd, 2)

			m.loginModel.usernameInput, cmds[0] = m.loginModel.usernameInput.Update(msg)
			m.loginModel.passwordInput, cmds[1] = m.loginModel.passwordInput.Update(msg)
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.loginModel.spinner, cmd = m.loginModel.spinner.Update(msg)
		return m, cmd

	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var b strings.Builder

	switch m.state {
	case loginState:
		b.WriteString(m.loginModel.usernameInput.View())
		b.WriteRune('\n')
		b.WriteString(m.loginModel.passwordInput.View())
		b.WriteRune('\n')

		button := &blurredButton
		if m.loginModel.focusIndex == 2 {
			button = &focusedButton
		}
		fmt.Fprintf(&b, "\n%s\n", *button)
	}

	if m.loginModel.isLoading {

		b.WriteString(fmt.Sprintf("%s Loading...\n", m.loginModel.spinner.View()))
	} else {
		b.WriteRune('\n')
	}

	b.WriteString(fmt.Sprintf("%s", m.loginModel.msg))
	return b.String()
}

func main() {
	if err := tea.NewProgram(initialModel()).Start(); err != nil {
		fmt.Printf("could not start program: %s\n", err)
		os.Exit(1)
	}
}
