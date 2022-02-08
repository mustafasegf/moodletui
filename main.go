package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/k3a/html2text"
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

func (m LoginModel) Move(s string, msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
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
		cmds = append(cmds, m.usernameInput.Focus())
	} else if m.focusIndex == int(passwordLoginState) {
		m.passwordInput.PromptStyle = focusedStyle
		m.passwordInput.TextStyle = focusedStyle
		cmds = append(cmds, m.passwordInput.Focus())
	}

	return m, tea.Batch(cmds...)
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
			return m.Move(s, msg)
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
		httpreq.token = token
		userID, err := httpreq.GetSceleId()

		if err != nil {
			m.msg = err.Error()
			return loginMsg{
				err: err,
			}
		}

		httpreq.userID = userID

		return loginMsg{
			err: nil,
		}
	}
}

type ForumModel struct {
	title string
	data  []Discussion
	page  int
}

func (m ForumModel) Init() tea.Cmd {
	return nil
}

func (m ForumModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()
		switch s {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		}
	}

	return m, tea.Batch(cmds...)
}

func (m ForumModel) View() string {
	var b strings.Builder
	b.WriteString(m.title)
	b.WriteRune('\n')

	for _, d := range m.data[:2] {
		b.WriteString(d.Name)
		b.WriteRune('\n')

		b.WriteString(html2text.HTML2Text(d.Message))
		b.WriteRune('\n')
		b.WriteRune('\n')
		b.WriteString("------------------------------\n")
	}

	return b.String()
}

func MakeForumModel(title string, forumid int) ForumModel {
	forum, _ := httpreq.GetForumDiscusstion(forumid, 0)
	m := ForumModel{
		title: title,
		data:  forum.Discussions,
		page:  0,
	}

	return m
}

//TODO: bikin submodel buat login, forum dll
type model struct {
	page    int
	state   state
	active  *tea.Model
	history []tea.Model
}

func (m model) Active() tea.Model {
	return m.history[int(m.page)]
}

func (m model) Next(newMdl tea.Model, state state) model {
	m.history = append(m.history, newMdl)
	m.page++
	m.state = state
	m.active = &newMdl
	return m
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
	err error
}

func initialModel() model {
	m := model{
		page:    0,
		history: make([]tea.Model, 0, 1),
	}

	if httpreq.token == "" || httpreq.userID == 0 {
		m.history = append(m.history, MakeLoginModel())
		m.state = loginState
		m.active = &m.history[0]
		return m
	}

	m.history = append(m.history, MakeForumModel("homepage", 1))
	m.state = forumState

	m.active = &m.history[0]
	return m
}

func (m model) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	cmds = append(cmds, m.Active().Init())
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case loginMsg:
		mdl := m.Active().(LoginModel)
		mdl.isLoading = false
		mdl.spinner.Finish()
		if msg.err != nil {
			mdl.msg = msg.err.Error()
		} else {
			mdl := MakeForumModel("homepage", 1)
			m = m.Next(mdl, forumState)
		}

		*m.active = mdl
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		s := msg.String()
		switch s {
		case "ctrl+c", "esc", "ctrl+q":
			return m, tea.Quit

		case "up", "down", "enter", "tab", "shift+tab":
			switch m.state {
			case loginState:
				mdl := m.Active().(LoginModel)
				cmds = make([]tea.Cmd, 1)
				*m.active, cmds[0] = mdl.Move(s, msg)

				return m, tea.Batch(cmds...)
			}
		}

	}

	switch m.state {
	case loginState:
		mdl := m.Active().(LoginModel)
		cmds = make([]tea.Cmd, 1)
		*m.active, cmds[0] = mdl.Update(msg)

		return m, tea.Batch(cmds...)
	}
	return m, tea.Batch(cmds...)
	// return m.Active().Update(msg)
}

func (m model) View() string {
	/* switch m.state {
	case loginState:
		return m.loginModel.View()
	}
	return "" */
	return m.Active().View()
}

type Config struct {
	Token  string `toml:"token"`
	UserID int    `toml:"userid"`
}

func LoadConfig() {
	_, err := os.OpenFile("config.toml", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		fmt.Println("can't open config")
		os.Exit(1)
	}

	var conf Config
	raw, err := ioutil.ReadFile("config.toml")
	if err != nil {
		fmt.Println("can't read config")
		os.Exit(1)
	}

	_, err = toml.Decode(string(raw), &conf)

	if err != nil {
		fmt.Println("config.toml corrupted")
		os.Exit(1)
	}
	httpreq.token = conf.Token
	httpreq.userID = conf.UserID
}

func main() {
	LoadConfig()
	if err := tea.NewProgram(initialModel()).Start(); err != nil {
		fmt.Printf("could not start program: %s\n", err)
		os.Exit(1)
	}
}
