/*
Package confirmation implements prompt for a binary confirmation such as a
yes/no question. It also offers customizable appreance and a customizable key
map.
*/
package confirmation

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"text/template"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/erikgeiser/promptkit"
	"github.com/muesli/termenv"
)

const (
	// DefaultTemplate defines the default appearance of the text input and can
	// be copied as a starting point for a custom template.
	DefaultTemplate = `
	{{- Bold .Prompt }}{{ " " -}}
	{{ if .YesSelected -}} ◉ {{- else -}} ○ {{- end -}}
	{{- print (Bold "Y") "es " -}}
	{{ if .NoSelected -}} ◉ {{- else -}} ○ {{- end -}}
	{{- print (Bold "N") "o" -}}
	`

	// DefaultConfirmationTemplate defines the default appearance with which the
	// finale result of the prompt is presented.
	DefaultConfirmationTemplate = `
	{{- print .Prompt " " -}}
	{{- if .FinalValue -}}
		{{- Foreground "32" "Yes" -}}
	{{- else -}}
		{{- Foreground "32" "No" -}}
	{{- end -}}
	{{- "\n" -}}
	`
)

// Value is the value of the confirmation prompt which can be Undecided, Yes or
// No.
type Value *bool

var (
	yes = true
	no  = false

	// Yes is a possible value of the confirmation prompt that corresponds to
	// true.
	Yes = Value(&yes)
	// No is a possible value of the confirmation prompt that corresponds to
	// false.
	No = Value(&no)
	// Undecided is a possible value of the confirmation prompt that is used
	// when neither Yes nor No are selected.
	Undecided = Value(nil)
)

// Confirmation represents a configurable confirmation prompt.
type Confirmation struct {
	// Prompt holds the question.
	Prompt string

	// DefaultValue decides if a value should already be selected at startup. By
	// default it it Undecided but it can be set to Yes (corresponds to true)
	// and No (corresponds to false).
	DefaultValue Value

	// Template holds the display template. A custom template can be used to
	// completely customize the appearance of the text input. If empty, the
	// DefaultTemplate is used. The following variables and functions are
	// available:
	//
	//  * Prompt string: The configured prompt.
	//  * YesSelected bool: Whether or not Yes is the currently selected
	//    value.
	//  * NoSelected bool: Whether or not No is the currently selected value.
	//  * Undecided bool: Whether or not Undecided is the currently selected
	//    value.
	//  * DefaultYes bool: Whether or not Yes is confiured as default value.
	//  * DefaultNo bool: Whether or not No is confiured as default value.
	//  * DefaultUndecided bool: Whether or not Undecided is confiured as
	//    default value.
	//  * TerminalWidth int: The width of the terminal.
	//  * promptkit.UtilFuncMap: Handy helper functions.
	//  * termenv TemplateFuncs (see https://github.com/muesli/termenv).
	//  * The functions specified in ExtendedTemplateScope.
	Template string

	// ConfirmationTemplate is rendered as soon as a input has been confirmed.
	// It is intended to permanently indicate the result of the prompt when the
	// input itself has disappeared. This template is only rendered in the Run()
	// method and NOT when the text input is used as a model. The following
	// variables and functions are available:
	//
	//  * FinalValue bool: The final value of the confirmation.
	//  * FinalValue string: The final value's string representation ("true"
	//    or "false").
	//  * Prompt string: The configured prompt.
	//  * DefaultYes bool: Whether or not Yes is confiured as default value.
	//  * DefaultNo bool: Whether or not No is confiured as default value.
	//  * DefaultUndecided bool: Whether or not Undecided is confiured as
	//    default value.
	//  * TerminalWidth int: The width of the terminal.
	//  * promptkit.UtilFuncMap: Handy helper functions.
	//  * termenv TemplateFuncs (see https://github.com/muesli/termenv).
	//  * The functions specified in ExtendedTemplateScope.
	ConfirmationTemplate string

	// ExtendedTemplateScope can be used to add additional functions to the
	// evaluation scope of the templates.
	ExtendedTemplateScope template.FuncMap

	// KeyMap determines with which keys the confirmation prompt is controlled.
	// By default, DefaultKeyMap is used.
	KeyMap *KeyMap

	// Output is the output writer, by default os.Stdout is used.
	Output io.Writer
	// Input is the input reader, by default, os.Stdin is used.
	Input io.Reader
}

// New creates a new text input.
func New(prompt string) *Confirmation {
	return &Confirmation{
		Prompt:                prompt,
		Template:              DefaultTemplate,
		ConfirmationTemplate:  DefaultConfirmationTemplate,
		KeyMap:                NewDefaultKeyMap(),
		DefaultValue:          Undecided,
		ExtendedTemplateScope: template.FuncMap{},
		Output:                os.Stdout,
		Input:                 os.Stdin,
	}
}

// RunPrompt executes the confirmation prompt.
func (c *Confirmation) RunPrompt() (bool, error) {
	tmpl, err := c.initConfirmationTemplate()
	if err != nil {
		return false, fmt.Errorf("initializing confirmation template: %w", err)
	}

	m := NewModel(c)

	p := tea.NewProgram(m, tea.WithOutput(c.Output), tea.WithInput(c.Input))
	if err := p.Start(); err != nil {
		return false, fmt.Errorf("running prompt: %w", err)
	}

	value, err := m.Value()
	if err != nil {
		return false, fmt.Errorf("reading value: %w", err)
	}

	if c.ConfirmationTemplate == "" {
		return value, nil
	}

	buffer := &bytes.Buffer{}

	err = tmpl.Execute(buffer, map[string]interface{}{
		"FinalValue":       value,
		"FinalValueString": fmt.Sprintf("%v", value),
		"Prompt":           m.Prompt,
		"DefaultYes":       m.DefaultValue == Yes,
		"DefaultNo":        m.DefaultValue == No,
		"DefaultUndecided": m.DefaultValue == Undecided,
		"TerminalWidth":    m.width,
	})
	if err != nil {
		return value, fmt.Errorf("execute confirmation template: %w", err)
	}

	_, err = fmt.Fprint(os.Stdout, promptkit.Wrap(buffer.String(), m.width))

	return value, err
}

func (c *Confirmation) initConfirmationTemplate() (*template.Template, error) {
	if c.ConfirmationTemplate == "" {
		return nil, nil
	}

	tmpl := template.New("confirmed")
	tmpl.Funcs(termenv.TemplateFuncs(termenv.ColorProfile()))
	tmpl.Funcs(promptkit.UtilFuncMap())
	tmpl.Funcs(c.ExtendedTemplateScope)

	return tmpl.Parse(c.ConfirmationTemplate)
}