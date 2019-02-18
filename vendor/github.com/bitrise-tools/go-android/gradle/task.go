package gradle

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/command"
)

// Task ...
type Task struct {
	name    string
	project Project
}

// GetVariants ...
func (task *Task) GetVariants() (Variants, error) {
	tasksOutput, err := getGradleOutput(task.project.location, "tasks", "--all")
	if err != nil {
		return nil, fmt.Errorf("%s, %s", tasksOutput, err)
	}
	return task.parseVariants(tasksOutput), nil
}

func (task *Task) parseVariants(gradleOutput string) Variants {
	//example gradleOutput:
	//"
	// lintMyflavorokStaging - Runs lint on the MyflavorokStaging build.
	// lintMyflavorRelease - Runs lint on the MyflavorRelease build.
	// lintVitalMyflavorRelease - Runs lint on the MyflavorRelease build.
	// lintMyflavorStaging - Runs lint on the MyflavorStaging build."
	tasks := Variants{}
lines:
	for _, l := range strings.Split(gradleOutput, "\n") {
		// l: " lintMyflavorokStaging - Runs lint on the MyflavorokStaging build."
		l = strings.TrimSpace(l)
		// l: "lintMyflavorokStaging - Runs lint on the MyflavorokStaging build."
		if l == "" {
			continue
		}
		// l: "lintMyflavorokStaging"
		l = strings.Split(l, " ")[0]
		var module string

		split := strings.Split(l, ":")
		if len(split) > 1 {
			module = split[0]
			l = split[1]
		}
		// module removed if any
		if strings.HasPrefix(l, task.name) {
			// task.name: "lint"
			// strings.HasPrefix will match lint and lintVital prefix also, we won't need lintVital so it is a conflict
			for _, conflict := range conflicts[task.name] {
				if strings.HasPrefix(l, conflict) {
					// if line has conflicting prefix don't do further checks with this line, skip...
					continue lines
				}
			}
			l = strings.TrimPrefix(l, task.name)
			// l: "MyflavorokStaging"
			if l == "" {
				continue
			}

			tasks[module] = append(tasks[module], l)
		}
	}

	for module, variants := range tasks {
		tasks[module] = cleanStringSlice(variants)
	}

	return tasks
}

func cleanModuleName(s string) string {
	if s == "" {
		return s
	}
	return ":" + s + ":"
}

// GetCommand ...
func (task *Task) GetCommand(v Variants, args ...string) *command.Model {
	var a []string
	for module, variants := range v {
		for _, variant := range variants {
			a = append(a, cleanModuleName(module)+task.name+variant)
		}
	}
	return command.NewWithStandardOuts(filepath.Join(task.project.location, "gradlew"), append(a, args...)...).
		SetDir(task.project.location)
}
