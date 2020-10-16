package main

import (
	"fmt"
	"os"
	"time"

	input "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"

	"github.com/yuzuy/todo-cli"
)

func main() {
	p := tea.NewProgram(initialize, update, view)
	if err := p.Start(); err != nil {
		fmt.Printf("there's been an error: %s\n", err.Error())
		os.Exit(1)
	}
}

var latestTaskID int

const (
	normalMode = iota
	doneTaskListMode
	additionalMode
	editMode
)

type model struct {
	cursor    int
	mode      int
	tasks     []*todo.Task
	doneTasks []*todo.Task

	newTaskNameModel  input.Model
	editTaskNameModel input.Model
}

func initialize() (tea.Model, tea.Cmd) {
	todos, doneTodos, ltID := loadTodos()
	latestTaskID = ltID

	cursor := 0
	if len(todos) != 0 {
		cursor = 1
	}

	newTaskNameModel := input.NewModel()
	newTaskNameModel.Placeholder = "New task name..."
	newTaskNameModel.Focus()
	editTaskNameModel := input.NewModel()
	editTaskNameModel.Focus()

	return model{
		cursor:            cursor,
		mode:              normalMode,
		tasks:             todos,
		doneTasks:         doneTodos,
		newTaskNameModel:  newTaskNameModel,
		editTaskNameModel: editTaskNameModel,
	}, nil
}

func update(msg tea.Msg, mdl tea.Model) (tea.Model, tea.Cmd) {
	m := mdl.(model)

	switch m.mode {
	case normalMode:
		return normalUpdate(msg, m)
	case doneTaskListMode:
		return doneTaskListUpdate(msg, m)
	case additionalMode:
		return additionalTaskUpdate(msg, m)
	case editMode:
		return editTaskUpdate(msg, m)
	}

	return m, nil
}

func normalUpdate(msg tea.Msg, m model) (tea.Msg, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j":
			if m.cursor < len(m.tasks) {
				m.cursor++
			}
		case "k":
			if m.cursor > 1 {
				m.cursor--
			}
		case "a":
			m.mode = additionalMode
			return m, input.Blink(m.newTaskNameModel)
		case "d":
			if m.cursor == 0 {
				break
			}
			m.tasks = append(m.tasks[:m.cursor-1], m.tasks[m.cursor:]...)
			if len(m.tasks) == 0 {
				m.cursor = 0
			} else {
				m.cursor = 1
			}
		case "e":
			if m.cursor == 0 {
				break
			}
			m.mode = editMode
			m.editTaskNameModel.Placeholder = m.tasks[m.cursor-1].Name
			return m, input.Blink(m.editTaskNameModel)
		case "x", "enter":
			if m.cursor == 0 {
				break
			}

			t := m.tasks[m.cursor-1]
			t.IsDone = true
			m.doneTasks = append(m.doneTasks, t)
			m.tasks = append(m.tasks[:m.cursor-1], m.tasks[m.cursor:]...)

			if len(m.tasks) == 0 {
				m.cursor = 0
			} else {
				m.cursor = 1
			}
		case "t":
			if m.mode == doneTaskListMode {
				if len(m.tasks) == 0 {
					m.cursor = 0
				} else {
					m.cursor = 1
				}
				m.mode = normalMode
				return m, nil
			}
			if len(m.doneTasks) == 0 {
				m.cursor = 0
			} else {
				m.cursor = 1
			}
			m.mode = doneTaskListMode
		case "q", "etc", "ctrl+c":
			storeTodos(m)
			return m, tea.Quit
		}
	}

	return m, nil
}

func doneTaskListUpdate(msg tea.Msg, m model) (tea.Msg, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j":
			if m.cursor < len(m.doneTasks) {
				m.cursor++
			}
		case "k":
			if m.cursor > 1 {
				m.cursor--
			}
		case "d":
			if m.cursor == 0 {
				break
			}
			m.doneTasks = append(m.tasks[:m.cursor-1], m.tasks[m.cursor:]...)
			if len(m.doneTasks) == 0 {
				m.cursor = 0
			} else {
				m.cursor = 1
			}
		case "x", "enter":
			if m.cursor == 0 {
				break
			}
			t := m.doneTasks[m.cursor-1]
			t.IsDone = false
			m.tasks = append(m.tasks, t)
			m.doneTasks = append(m.doneTasks[:m.cursor-1], m.doneTasks[m.cursor:]...)
			if len(m.doneTasks) == 0 {
				m.cursor = 0
			} else {
				m.cursor = 1
			}
		case "t":
			if len(m.tasks) == 0 {
				m.cursor = 0
			} else {
				m.cursor = 1
			}
			m.mode = normalMode
		case "q", "ctrl+c":
			storeTodos(m)
			return m, tea.Quit
		}
	}

	return m, nil
}

func additionalTaskUpdate(msg tea.Msg, m model) (tea.Msg, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "etc", "q":
			m.mode = normalMode
			m.newTaskNameModel.Reset()
			return m, nil
		case "enter":
			if m.newTaskNameModel.Value() == "" {
				return m, nil
			}

			m.tasks = append(m.tasks, &todo.Task{
				ID:        latestTaskID + 1,
				Name:      m.newTaskNameModel.Value(),
				CreatedAt: time.Now(),
			})
			latestTaskID++

			m.cursor++
			m.mode = normalMode
			m.newTaskNameModel.Reset()
			return m, nil
		}
	}

	m.newTaskNameModel, cmd = input.Update(msg, m.newTaskNameModel)

	return m, cmd
}

func editTaskUpdate(msg tea.Msg, m model) (tea.Msg, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "etc", "q":
			m.mode = normalMode
			m.editTaskNameModel.Reset()
			return m, nil
		case "enter":
			if m.editTaskNameModel.Value() == "" {
				return m, nil
			}
			m.tasks[m.cursor-1].Name = m.editTaskNameModel.Value()

			m.mode = normalMode
			m.editTaskNameModel.Reset()
			return m, nil
		}
	}

	m.editTaskNameModel, cmd = input.Update(msg, m.editTaskNameModel)

	return m, cmd
}

func view(mdl tea.Model) string {
	m := mdl.(model)

	switch m.mode {
	case normalMode, doneTaskListMode:
		return normalView(m)
	case additionalMode:
		return additionalTaskView(m)
	case editMode:
		return editTaskView(m)
	}

	return ""
}

func normalView(m model) string {
	var s string
	var title termenv.Style
	var tasksToDisplay []*todo.Task
	switch m.mode {
	case normalMode:
		if len(m.tasks) == 0 {
			return "You have no tasks. Press 'a' to add your task!\n"
		}
		title = termenv.String("YOUR TASKS")
		tasksToDisplay = m.tasks
	case doneTaskListMode:
		if len(m.doneTasks) == 0 {
			return "You have no done tasks.\n"
		}
		title = termenv.String("YOUR DONE TASKS")
		tasksToDisplay = m.doneTasks
	}
	title = title.Bold().Underline()
	s = fmt.Sprintf("%v\n\n", title)

	for i, v := range tasksToDisplay {
		cursor := termenv.String(" ")
		if m.cursor == i+1 {
			cursor = termenv.String(">").Foreground(termenv.ANSIYellow)
		}
		taskName := termenv.String(v.Name)
		taskName = taskName.Bold()
		timeLayout := "2006-01-02 15:04"

		s += fmt.Sprintf("%v #%d: %s (%s)\n", cursor, v.ID, taskName, v.CreatedAt.Format(timeLayout))
	}

	return s
}

func additionalTaskView(m model) string {
	title := termenv.String("Additional Mode").Bold().Underline()
	return fmt.Sprintf("%v\n\nInput the new task name\n\n%s\n", title, input.View(m.newTaskNameModel))
}

func editTaskView(m model) string {
	title := termenv.String("Edit Mode").Bold().Underline()
	return fmt.Sprintf("%v\n\nInput the new task name\n\n%s\n", title, input.View(m.editTaskNameModel))
}

func report(err error) {
	fmt.Printf("todo-cli: %s\n", err.Error())
	os.Exit(1)
}
