package note

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/mickael-menu/zk/core/templ"
	"github.com/mickael-menu/zk/core/zk"
	"github.com/mickael-menu/zk/util/date"
	"github.com/mickael-menu/zk/util/errors"
	"github.com/mickael-menu/zk/util/opt"
	"github.com/mickael-menu/zk/util/os"
	"github.com/mickael-menu/zk/util/paths"
	"github.com/mickael-menu/zk/util/rand"
)

// CreateOpts holds the options to create a new note.
type CreateOpts struct {
	// Current configuration.
	Config zk.Config
	// Parent directory for the new note.
	Dir zk.Dir
	// Title of the note.
	Title opt.String
	// Initial content of the note, which will be injected in the template.
	Content opt.String
}

// ErrNoteExists is an error returned when a note already exists with the
// filename generated by Create().
type ErrNoteExists struct {
	Name string
	Path string
}

func (e ErrNoteExists) Error() string {
	return fmt.Sprintf("%s: note already exists", e.Path)
}

// Create generates a new note from the given options.
// Returns the path of the newly created note.
func Create(
	opts CreateOpts,
	templateLoader templ.Loader,
	date date.Provider,
) (string, error) {
	wrap := errors.Wrapperf("new note")

	filenameTemplate, err := templateLoader.Load(opts.Dir.Config.Note.FilenameTemplate)
	if err != nil {
		return "", err
	}

	var bodyTemplate templ.Renderer = templ.NullRenderer
	if templatePath := opts.Dir.Config.Note.BodyTemplatePath.Unwrap(); templatePath != "" {
		absPath, ok := opts.Config.LocateTemplate(templatePath)
		if !ok {
			return "", wrap(fmt.Errorf("%s: cannot find template", templatePath))
		}
		bodyTemplate, err = templateLoader.LoadFile(absPath)
		if err != nil {
			return "", wrap(err)
		}
	}

	createdNote, err := create(opts, createDeps{
		filenameTemplate: filenameTemplate,
		bodyTemplate:     bodyTemplate,
		genId:            rand.NewIDGenerator(opts.Dir.Config.Note.IDOptions),
		validatePath:     validatePath,
		now:              date.Date(),
	})
	if err != nil {
		return "", wrap(err)
	}

	err = paths.WriteString(createdNote.path, createdNote.content)
	if err != nil {
		return "", wrap(err)
	}

	return createdNote.path, nil
}

func validatePath(path string) (bool, error) {
	exists, err := paths.Exists(path)
	return !exists, err
}

type createdNote struct {
	path    string
	content string
}

// renderContext holds the placeholder values which will be expanded in the templates.
type renderContext struct {
	ID           string `handlebars:"id"`
	Title        string
	Content      string
	Dir          string
	Filename     string
	FilenameStem string `handlebars:"filename-stem"`
	Extra        map[string]string
	Now          time.Time
	Env          map[string]string
}

type createDeps struct {
	filenameTemplate templ.Renderer
	bodyTemplate     templ.Renderer
	genId            func() string
	validatePath     func(path string) (bool, error)
	now              time.Time
}

func create(
	opts CreateOpts,
	deps createDeps,
) (*createdNote, error) {
	context := renderContext{
		Title:   opts.Title.OrString(opts.Dir.Config.Note.DefaultTitle).Unwrap(),
		Content: opts.Content.Unwrap(),
		Dir:     opts.Dir.Name,
		Extra:   opts.Dir.Config.Extra,
		Now:     deps.now,
		Env:     os.Env(),
	}

	path, context, err := genPath(context, opts.Dir, deps)
	if err != nil {
		return nil, err
	}

	content, err := deps.bodyTemplate.Render(context)
	if err != nil {
		return nil, err
	}

	return &createdNote{path: path, content: content}, nil
}

func genPath(
	context renderContext,
	dir zk.Dir,
	deps createDeps,
) (string, renderContext, error) {
	var err error
	var filename string
	var path string
	for i := 0; i < 50; i++ {
		context.ID = deps.genId()

		filename, err = deps.filenameTemplate.Render(context)
		if err != nil {
			return "", context, err
		}

		filename = filename + "." + dir.Config.Note.Extension
		path = filepath.Join(dir.Path, filename)
		validPath, err := deps.validatePath(path)
		if err != nil {
			return "", context, err
		} else if validPath {
			context.Filename = filepath.Base(path)
			context.FilenameStem = paths.FilenameStem(path)
			return path, context, nil
		}
	}

	return "", context, ErrNoteExists{
		Name: filepath.Join(dir.Name, filename),
		Path: path,
	}
}
