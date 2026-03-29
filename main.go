package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	Column      string    `json:"column"`
	Labels      []string  `json:"labels"`
	CreatedAt   time.Time `json:"created_at"`
}

type BoardConfig struct {
	Columns []string `json:"columns"`
}

type TaskList struct {
	Tasks  []Task      `json:"tasks"`
	Config BoardConfig `json:"config"`
}

type ViewMode int

const (
	ViewBoard ViewMode = iota
	ViewAdd
	ViewEdit
	ViewHelp
)


var (
	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FBBF24")).
		Background(lipgloss.Color("#1F2937")).
		Padding(0, 2)

	columnBase = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 1).
		Width(28).
		Height(18)

	selectedColumn = columnBase.Copy().
		BorderForeground(lipgloss.Color("#FBBF24"))

	taskStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(0, 1).
		MarginBottom(1)

	selectedTask = taskStyle.Copy().
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("#FBBF24")).
		Bold(true)

	completedTask = taskStyle.Copy().
		Foreground(lipgloss.Color("#6B7280")).
		Strikethrough(true)
)

// assign colors to columns dynamically
func colColor(name string) lipgloss.Color {
	switch name {
	case "todo":
		return "#3B82F6"
	case "doing":
		return "#F59E0B"
	case "done":
		return "#10B981"
	case "bug":
		return "#EF4444"
	case "feature":
		return "#8B5CF6"
	case "refactor":
		return "#EC4899"
	default:
		return "#9CA3AF"
	}
}

func (m model) View() string {
	if m.mode == ViewAdd {
		return headerStyle.Render(" ADD TASK ") + "\n\n" +
			m.textarea.View() +
			"\n\nenter = save • esc = cancel"
	}

	if m.mode == ViewEdit {
		return headerStyle.Render(" EDIT TASK ") + "\n\n" +
			m.textarea.View() +
			"\n\nenter = save • esc = cancel"
	}

	var cols []string

	for i, col := range m.columns {
		cols = append(cols, m.renderColumn(col, i))
	}

	header := headerStyle.Render("  🧺 BASKET  ")

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Render("h/l move • j/k select • n new • e edit • d delete • m move • space done • q quit")

	return header + "\n\n" +
		lipgloss.JoinHorizontal(lipgloss.Top, cols...) +
		"\n\n" + footer
}

func (m model) renderColumn(col string, index int) string {
	color := colColor(col)
	tasks := m.getTasks(col)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(color).
		Align(lipgloss.Center).
		Width(24).
		Render(strings.ToUpper(col))

	var items []string

	for i, t := range tasks {
		items = append(items, m.renderTask(t, index == m.selectedCol && i == m.selectedTask))
	}

	if len(items) == 0 {
		items = append(items,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4B5563")).
				Italic(true).
				Render("No tasks"),
		)
	}

	content := title + "\n" + strings.Repeat("─", 24) + "\n\n" +
		strings.Join(items, "\n")

	style := columnBase
	if index == m.selectedCol {
		style = selectedColumn
	}

	return style.BorderForeground(color).Render(content)
}

func (m model) renderTask(t *Task, selected bool) string {
	checkbox := "☐"
	if t.Completed {
		checkbox = "☑"
	}

	title := t.Title
	if len(title) > 20 {
		title = title[:17] + "..."
	}

	labelText := ""
	if len(t.Labels) > 0 {
		labelText = "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Render("#" + strings.Join(t.Labels, " #"))
	}

	content := fmt.Sprintf("%s %s%s", checkbox, title, labelText)

	style := taskStyle
	if selected {
		style = selectedTask
	} else if t.Completed {
		style = completedTask
	}

	return style.Width(24).Render(content)
}


type model struct {
	tasks        []Task
	columns      []string
	selectedCol  int
	selectedTask int
	mode         ViewMode
	textarea     textarea.Model
	editingTask  *Task
	width        int
	height       int
	path         string
}

var columnStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	Padding(1, 2).
	Width(30).
	Height(20)

var selectedColumnStyle = columnStyle.Copy().
	BorderForeground(lipgloss.Color("#FBBF24"))

var selectedTaskStyle = lipgloss.NewStyle().
	Border(lipgloss.ThickBorder()).
	BorderForeground(lipgloss.Color("#FBBF24")).
	Padding(0, 1)

func defaultColumns() []string {
	return []string{"todo", "doing", "done", "feature", "bug", "refactor"}
}
func getPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ".basket.json"
	}
	return filepath.Join(cwd, ".basket.json")
}

func getLocalPath() string {
	return getPath()
}
func load(path string) ([]Task, []string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return []Task{}, defaultColumns()
	}

	var t TaskList
	if err := json.Unmarshal(data, &t); err != nil {
		return []Task{}, defaultColumns()
	}

	if len(t.Config.Columns) == 0 {
		t.Config.Columns = defaultColumns()
	}

	return t.Tasks, t.Config.Columns
}

func save(path string, tasks []Task, cols []string) {
	data, _ := json.MarshalIndent(TaskList{
		Tasks: tasks,
		Config: BoardConfig{
			Columns: cols,
		},
	}, "", "  ")

	_ = os.WriteFile(path, data, 0644)
}
func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "New task..."
	ta.Focus()
	ta.SetWidth(50)
	ta.SetHeight(3)
	ta.CharLimit = 500

	path := getPath()
	tasks, cols := load(path)

	return model{
		tasks:   tasks,
		columns: cols,
		path:    path,
		mode:    ViewBoard,
		textarea: ta, // ✅ IMPORTANT
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch m.mode {

		case ViewBoard:
			switch msg.String() {

			case "q":
				return m, tea.Quit

			case "h":
				if m.selectedCol > 0 {
					m.selectedCol--
				}

			case "l":
				if m.selectedCol < len(m.columns)-1 {
					m.selectedCol++
				}

			case "j":
				m.selectedTask++

			case "k":
				if m.selectedTask > 0 {
					m.selectedTask--
				}

			case "n":
				m.mode = ViewAdd
				return m, m.textarea.Focus()

			case " ":
				task := m.getSelectedTask()
				if task != nil {
					task.Completed = !task.Completed
					m.save()
				}

			case "m":
				task := m.getSelectedTask()
				if task != nil {
					idx := m.selectedCol + 1
					if idx >= len(m.columns) {
						idx = 0
					}
					task.Column = m.columns[idx]
					m.save()
				}

			case "d":
				m.deleteSelected()

			case "e":
				task := m.getSelectedTask()
				if task != nil {
					m.mode = ViewEdit
					m.editingTask = task
					m.textarea.SetValue(task.Description)
					return m, m.textarea.Focus()
				}
			}

		case ViewAdd:
			switch msg.String() {
			case "esc":
				m.mode = ViewBoard
			case "enter":
				title := strings.TrimSpace(m.textarea.Value())
				if title != "" {
					m.tasks = append(m.tasks, Task{
						ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
						Title:     title,
						Column:    m.columns[m.selectedCol],
						CreatedAt: time.Now(),
					})
					m.save()
				}
				m.textarea.Reset()
				m.mode = ViewBoard
			}
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd

		case ViewEdit:
			switch msg.String() {
			case "esc":
				m.mode = ViewBoard
			case "enter":
				if m.editingTask != nil {
					m.editingTask.Description = m.textarea.Value()
					m.save()
				}
				m.mode = ViewBoard
			}
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *model) save() {
	save(m.path, m.tasks, m.columns)
}

func (m *model) getTasks(col string) []*Task {
	var out []*Task
	for i := range m.tasks {
		if m.tasks[i].Column == col {
			out = append(out, &m.tasks[i])
		}
	}
	return out
}

func (m *model) getSelectedTask() *Task {
	col := m.columns[m.selectedCol]
	tasks := m.getTasks(col)
	if len(tasks) == 0 {
		return nil
	}
	if m.selectedTask >= len(tasks) {
		m.selectedTask = len(tasks) - 1
	}
	return tasks[m.selectedTask]
}

func (m *model) deleteSelected() {
	col := m.columns[m.selectedCol]
	tasks := m.getTasks(col)
	if len(tasks) == 0 {
		return
	}
	id := tasks[m.selectedTask].ID

	for i := range m.tasks {
		if m.tasks[i].ID == id {
			m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
			break
		}
	}
	m.save()
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}
