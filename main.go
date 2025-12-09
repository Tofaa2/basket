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

// Priority levels
type Priority int

const (
	PriorityLowest Priority = iota
	PriorityLow
	PriorityMedium
	PriorityHigh
	PriorityHighest
)

func (p Priority) String() string {
	switch p {
	case PriorityLowest:
		return "LOWEST"
	case PriorityLow:
		return "LOW"
	case PriorityMedium:
		return "MEDIUM"
	case PriorityHigh:
		return "HIGH"
	case PriorityHighest:
		return "HIGHEST"
	default:
		return "MEDIUM"
	}
}

func (p Priority) Color() lipgloss.Color {
	switch p {
	case PriorityLowest:
		return lipgloss.Color("#6B7280")
	case PriorityLow:
		return lipgloss.Color("#3B82F6")
	case PriorityMedium:
		return lipgloss.Color("#8B5CF6")
	case PriorityHigh:
		return lipgloss.Color("#F59E0B")
	case PriorityHighest:
		return lipgloss.Color("#EF4444")
	default:
		return lipgloss.Color("#8B5CF6")
	}
}

// Task represents a single task
type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	Priority    Priority  `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
}

// TaskList holds tasks
type TaskList struct {
	Tasks []Task `json:"tasks"`
}

// ViewMode represents the current view
type ViewMode int

const (
	ViewBoard ViewMode = iota
	ViewAdd
	ViewEdit
	ViewHelp
)

type model struct {
	tasks           []Task
	globalTasks     []Task
	localTasks      []Task
	selectedCol     int // which priority column
	selectedTask    int // which task in that column
	scrollOffset    int // scroll offset for tasks in column
	colScrollOffset int // horizontal scroll offset for columns
	mode            ViewMode
	showingLocal    bool
	textarea        textarea.Model
	editingTask     *Task
	width           int
	height          int
	globalPath      string
	localPath       string
	hasLocal        bool
}

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FBBF24")).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 2)

	columnStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			Width(30).
			Height(20)

	selectedColumnStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FBBF24")).
				Padding(1, 2).
				Width(30).
				Height(20)

	taskCardStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			Padding(0, 1).
			MarginBottom(1)

	selectedTaskStyle = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color("#FBBF24")).
				Padding(0, 1).
				MarginBottom(1).
				Bold(true)

	completedTaskStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				Padding(0, 1).
				MarginBottom(1).
				Foreground(lipgloss.Color("#6B7280")).
				Strikethrough(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FBBF24")).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 2).
			MarginBottom(1)
)

func getGlobalTasksPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "basket-tasks.json"
	}
	return filepath.Join(home, "basket-tasks.json")
}

func getLocalTasksPath() (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	path := filepath.Join(cwd, ".basket.json")
	if _, err := os.Stat(path); err == nil {
		return path, true
	}
	return path, false
}

func loadTasks(path string) ([]Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Task{}, nil
		}
		return nil, err
	}

	var taskList TaskList
	if err := json.Unmarshal(data, &taskList); err != nil {
		return nil, err
	}
	return taskList.Tasks, nil
}

func saveTasks(path string, tasks []Task) error {
	taskList := TaskList{Tasks: tasks}
	data, err := json.MarshalIndent(taskList, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Enter task title..."
	ta.Focus()
	ta.CharLimit = 500
	ta.SetWidth(60)
	ta.SetHeight(3)

	globalPath := getGlobalTasksPath()
	localPath, hasLocal := getLocalTasksPath()

	globalTasks, _ := loadTasks(globalPath)
	var localTasks []Task
	if hasLocal {
		localTasks, _ = loadTasks(localPath)
	}

	tasks := localTasks
	showingLocal := true

	if !hasLocal || len(localTasks) == 0 {
		tasks = globalTasks
		showingLocal = false
		if !hasLocal {
			localTasks = []Task{}
		}
	}

	return model{
		tasks:        tasks,
		globalTasks:  globalTasks,
		localTasks:   localTasks,
		mode:         ViewBoard,
		showingLocal: showingLocal,
		textarea:     ta,
		globalPath:   globalPath,
		localPath:    localPath,
		hasLocal:     hasLocal,
		selectedCol:  2, // Start at MEDIUM
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case ViewBoard:
			return m.updateBoard(msg)
		case ViewAdd:
			return m.updateAdd(msg)
		case ViewEdit:
			return m.updateEdit(msg)
		case ViewHelp:
			if msg.String() == "esc" || msg.String() == "q" {
				m.mode = ViewBoard
			}
			return m, nil
		}
	}

	return m, nil
}

func (m model) updateBoard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "left", "h":
		if m.selectedCol > 0 {
			m.selectedCol--
		} else {
			m.selectedCol = 4
		}
		tasksInNewCol := m.getTasksInColumn(Priority(m.selectedCol))
		if len(tasksInNewCol) == 0 {
			m.selectedTask = 0
		} else if m.selectedTask >= len(tasksInNewCol) {
			m.selectedTask = len(tasksInNewCol) - 1
		}
		m.updateHorizontalScroll()

	case "right", "l":
		if m.selectedCol < 4 {
			m.selectedCol++
		} else {
			m.selectedCol = 0
		}
		tasksInNewCol := m.getTasksInColumn(Priority(m.selectedCol))
		if len(tasksInNewCol) == 0 {
			m.selectedTask = 0
		} else if m.selectedTask >= len(tasksInNewCol) {
			m.selectedTask = len(tasksInNewCol) - 1
		}
		m.updateHorizontalScroll()

	case "up", "k":
		tasksInCol := m.getTasksInColumn(Priority(m.selectedCol))
		if m.selectedTask > 0 && len(tasksInCol) > 0 {
			m.selectedTask--
			// Scroll up if needed
			if m.selectedTask < m.scrollOffset {
				m.scrollOffset = m.selectedTask
			}
		}

	case "down", "j":
		tasksInCol := m.getTasksInColumn(Priority(m.selectedCol))
		if len(tasksInCol) > 0 && m.selectedTask < len(tasksInCol)-1 {
			m.selectedTask++
			maxVisible := 8
			if m.selectedTask >= m.scrollOffset+maxVisible {
				m.scrollOffset = m.selectedTask - maxVisible + 1
			}
		}

	case " ", "enter":
		tasksInCol := m.getTasksInColumn(Priority(m.selectedCol))
		if len(tasksInCol) > 0 && m.selectedTask < len(tasksInCol) {
			for i := range m.tasks {
				if m.tasks[i].ID == tasksInCol[m.selectedTask].ID {
					m.tasks[i].Completed = !m.tasks[i].Completed
					m.saveCurrent()
					break
				}
			}
		}

	case "n":
		m.mode = ViewAdd
		m.textarea.Reset()
		m.textarea.Placeholder = "Enter task title..."
		m.textarea.SetHeight(3)
		return m, m.textarea.Focus()

	case "e":
		tasksInCol := m.getTasksInColumn(Priority(m.selectedCol))
		if len(tasksInCol) > 0 && m.selectedTask < len(tasksInCol) {
			for i := range m.tasks {
				if m.tasks[i].ID == tasksInCol[m.selectedTask].ID {
					m.mode = ViewEdit
					m.editingTask = &m.tasks[i]
					m.textarea.SetValue(m.editingTask.Description)
					m.textarea.Placeholder = "Enter task description..."
					m.textarea.SetHeight(10)
					return m, m.textarea.Focus()
				}
			}
		}

	case "d":
		tasksInCol := m.getTasksInColumn(Priority(m.selectedCol))
		if len(tasksInCol) > 0 && m.selectedTask < len(tasksInCol) {
			taskID := tasksInCol[m.selectedTask].ID
			for i := range m.tasks {
				if m.tasks[i].ID == taskID {
					m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
					if m.selectedTask >= len(m.getTasksInColumn(Priority(m.selectedCol))) && m.selectedTask > 0 {
						m.selectedTask--
					}
					m.saveCurrent()
					break
				}
			}
		}

	case "m":
		tasksInCol := m.getTasksInColumn(Priority(m.selectedCol))
		if len(tasksInCol) > 0 && m.selectedTask < len(tasksInCol) {
			for i := range m.tasks {
				if m.tasks[i].ID == tasksInCol[m.selectedTask].ID {
					newPriority := (m.tasks[i].Priority + 1) % 5
					m.tasks[i].Priority = newPriority
					m.saveCurrent()

					m.selectedCol = int(newPriority)
					tasksInNewCol := m.getTasksInColumn(newPriority)
					for idx, task := range tasksInNewCol {
						if task.ID == m.tasks[i].ID {
							m.selectedTask = idx
							break
						}
					}
					break
				}
			}
		}

	case "t":
		if m.hasLocal {
			m.showingLocal = !m.showingLocal
			if m.showingLocal {
				m.tasks = m.localTasks
			} else {
				m.tasks = m.globalTasks
			}
			// Reset position
			m.selectedCol = 2
			m.selectedTask = 0
			m.scrollOffset = 0
			m.colScrollOffset = 0
			m.updateHorizontalScroll()
		} else {
			// If no local file exists, create it by switching to local mode
			m.showingLocal = true
			m.hasLocal = true
			m.localTasks = []Task{}
			m.tasks = m.localTasks
			m.selectedCol = 2
			m.selectedTask = 0
			m.scrollOffset = 0
			m.colScrollOffset = 0
			m.updateHorizontalScroll()
		}

	case "?":
		m.mode = ViewHelp
	}

	return m, nil
}

func (m model) updateAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		m.mode = ViewBoard
		return m, nil

	case "ctrl+s":
		title := strings.TrimSpace(m.textarea.Value())
		if title != "" {
			newTask := Task{
				ID:        generateID(),
				Title:     title,
				Priority:  Priority(m.selectedCol),
				CreatedAt: time.Now(),
			}
			m.tasks = append(m.tasks, newTask)
			m.saveCurrent()
		}
		m.mode = ViewBoard
		return m, nil
	}

	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m model) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		m.mode = ViewBoard
		m.editingTask = nil
		return m, nil

	case "ctrl+s":
		if m.editingTask != nil {
			m.editingTask.Description = strings.TrimSpace(m.textarea.Value())
			m.saveCurrent()
		}
		m.mode = ViewBoard
		m.editingTask = nil
		return m, nil
	}

	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m *model) saveCurrent() {
	if m.showingLocal {
		m.localTasks = make([]Task, len(m.tasks))
		copy(m.localTasks, m.tasks)
		if !m.hasLocal {
			m.hasLocal = true
		}
		saveTasks(m.localPath, m.localTasks)
	} else {
		m.globalTasks = make([]Task, len(m.tasks))
		copy(m.globalTasks, m.tasks)
		saveTasks(m.globalPath, m.globalTasks)
	}
}

func (m model) getTasksInColumn(priority Priority) []Task {
	var tasks []Task
	for _, task := range m.tasks {
		if task.Priority == priority {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

func (m *model) updateHorizontalScroll() {
	visibleCols := 3

	desiredScroll := m.selectedCol - (visibleCols / 2)

	if desiredScroll < 0 {
		m.colScrollOffset = 0
	} else if desiredScroll > 5-visibleCols {
		m.colScrollOffset = 5 - visibleCols
	} else {
		m.colScrollOffset = desiredScroll
	}
}

func (m model) getVisibleColumns() (int, int) {
	if m.width >= 160 {
		return 0, 5
	} else if m.width >= 128 {
		start := m.colScrollOffset
		if start > 1 {
			start = 1
		}
		return start, start + 4
	} else {
		start := m.colScrollOffset
		if start > 2 {
			start = 2
		}
		return start, start + 3
	}
}

func (m model) View() string {
	switch m.mode {
	case ViewAdd:
		return m.viewAdd()
	case ViewEdit:
		return m.viewEdit()
	case ViewHelp:
		return m.viewHelp()
	default:
		return m.viewBoard()
	}
}

func (m model) viewBoard() string {
	var b strings.Builder

	// Header
	source := "🌍 GLOBAL"
	if m.showingLocal {
		source = "📂 LOCAL"
	}
	header := headerStyle.Render(fmt.Sprintf("  🧺 BASKET  %s  ", source))
	b.WriteString(header + "\n\n")

	startCol, endCol := m.getVisibleColumns()
	priorities := []Priority{PriorityLowest, PriorityLow, PriorityMedium, PriorityHigh, PriorityHighest}

	var visibleColumns []string
	for i := startCol; i < endCol && i < 5; i++ {
		priority := priorities[i]
		column := m.renderColumn(priority, i == m.selectedCol)
		visibleColumns = append(visibleColumns, column)
	}

	var columnsWithIndicators []string

	if startCol > 0 {
		leftIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FBBF24")).
			Bold(true).
			Render("◀")
		columnsWithIndicators = append(columnsWithIndicators, leftIndicator)
	}

	columnsWithIndicators = append(columnsWithIndicators, visibleColumns...)

	if endCol < 5 {
		rightIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FBBF24")).
			Bold(true).
			Render("▶")
		columnsWithIndicators = append(columnsWithIndicators, rightIndicator)
	}

	columnsJoined := lipgloss.JoinHorizontal(lipgloss.Top, columnsWithIndicators...)
	b.WriteString(columnsJoined + "\n\n")

	help := helpStyle.Render("h/l columns • j/k tasks • space toggle • m move • n new • e edit • d delete • t switch • ? help • q quit")
	b.WriteString(help)

	return b.String()
}

func (m model) renderColumn(priority Priority, isSelected bool) string {
	var b strings.Builder

	headerText := priority.String()
	if isSelected {
		headerText = "▶ " + headerText + " ◀"
	}
	colHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(priority.Color()).
		Width(26).
		Align(lipgloss.Center).
		Render(headerText)

	b.WriteString(colHeader + "\n")

	separator := strings.Repeat("─", 26)
	if isSelected {
		separator = strings.Repeat("═", 26)
	}
	sepStyle := lipgloss.NewStyle()
	if isSelected {
		sepStyle = sepStyle.Foreground(priority.Color())
	}
	b.WriteString(sepStyle.Render(separator) + "\n\n")

	tasks := m.getTasksInColumn(priority)

	if len(tasks) == 0 {
		emptyText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4B5563")).
			Italic(true).
			Render("No tasks")
		b.WriteString(emptyText + "\n")
	} else {
		maxVisible := 8
		start := 0
		end := len(tasks)

		if isSelected {
			start = m.scrollOffset
			end = m.scrollOffset + maxVisible
			if end > len(tasks) {
				end = len(tasks)
			}
		}

		if isSelected && start > 0 {
			indicator := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9CA3AF")).
				Render("    ▲ more above")
			b.WriteString(indicator + "\n")
		}

		for i := start; i < end; i++ {
			task := tasks[i]
			isTaskSelected := isSelected && i == m.selectedTask
			b.WriteString(m.renderTask(task, isTaskSelected) + "\n")
		}

		if isSelected && end < len(tasks) {
			indicator := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9CA3AF")).
				Render("    ▼ more below")
			b.WriteString(indicator + "\n")
		}
	}

	content := b.String()

	style := columnStyle
	if isSelected {
		style = selectedColumnStyle
	}
	style = style.BorderForeground(priority.Color())

	return style.Render(content)
}

func (m model) renderTask(task Task, isSelected bool) string {
	var b strings.Builder

	checkbox := "☐"
	if task.Completed {
		checkbox = "☑"
	}

	title := task.Title
	if len(title) > 20 {
		title = title[:17] + "..."
	}

	content := fmt.Sprintf("%s %s", checkbox, title)

	style := taskCardStyle
	if isSelected {
		style = selectedTaskStyle
	} else if task.Completed {
		style = completedTaskStyle
	}

	b.WriteString(style.Width(22).Render(content))

	return b.String()
}

func (m model) viewAdd() string {
	priorityName := Priority(m.selectedCol).String()
	priorityColor := Priority(m.selectedCol).Color()

	titleText := fmt.Sprintf("📝 ADD TASK TO %s", priorityName)
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(priorityColor).
		Render(titleText)

	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		title,
		m.textarea.View(),
		helpStyle.Render("ctrl+s to save • esc to cancel"),
	)
}

func (m model) viewEdit() string {
	title := "✏️  EDIT TASK"
	if m.editingTask != nil {
		taskTitle := m.editingTask.Title
		if len(taskTitle) > 40 {
			taskTitle = taskTitle[:37] + "..."
		}
		title = fmt.Sprintf("✏️  %s", taskTitle)
	}

	styledTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FBBF24")).
		Render(title)

	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		styledTitle,
		m.textarea.View(),
		helpStyle.Render("ctrl+s to save • esc to cancel"),
	)
}

func (m model) viewHelp() string {
	help := `
╔═══════════════════════════════════════╗
║          🧺 BASKET HELP               ║
╚═══════════════════════════════════════╝

NAVIGATION
  h/←  Move to left column
  l/→  Move to right column  
  k/↑  Move up in column
  j/↓  Move down in column

TASK ACTIONS
  space    Toggle completion
  m        Move task to next priority
  n        Add new task
  e        Edit task description
  d        Delete task

VIEW
  t        Switch global/local
  ?        Show this help
  q        Quit

STORAGE
  Global   ~/basket-tasks.json
  Local    ./.basket.json

Priority columns from left to right:
  LOWEST → LOW → MEDIUM → HIGH → HIGHEST

Press ESC or q to return
`
	return help
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
