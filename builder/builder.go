package builder

import (
	"errors"
	"github.com/Unknwon/com"
	"gopkg.in/inconshreveable/log15.v2"
	"os"
	"pugo/parser"
	"pugo/render"
	"time"
)

var (
	ErrSrcDirMissing = errors.New("builder-src-dir-missing")
	ErrTplDirMissing = errors.New("builder-tpl-dir-missing")
)

type Builder struct {
	srcDir     string
	tplDir     string
	isBuilding bool

	renders *render.Renders
	report  *Report
	parser  parser.Parser

	Error error
}

func New(sourceDir, templateDir, currentTheme string, debug bool) *Builder {
	if !com.IsDir(sourceDir) {
		return &Builder{Error: ErrSrcDirMissing}
	}
	if !com.IsDir(templateDir) {
		return &Builder{Error: ErrTplDirMissing}
	}
	builder := &Builder{
		srcDir: sourceDir,
		tplDir: templateDir,
		parser: parser.NewCommonParser(),
	}
	r, err := render.NewRenders(templateDir, currentTheme, debug)
	if err != nil {
		return &Builder{Error: err}
	}
	builder.renders = r
	return builder
}

func (b *Builder) Renders() *render.Renders {
	return b.renders
}

func (b *Builder) Build(dest string) {
	// if on build, do not again
	if b.isBuilding {
		return
	}
	log15.Debug("Build.Start")
	r := &Report{
		Begin: time.Now(),
	}
	if err := os.MkdirAll(dest, os.ModePerm); err != nil {
		r.Error = err
		b.report = r
		return
	}
	ctx := &context{
		DstDir: dest,
	}
	b.isBuilding = true
	b.nav(ctx, r)
	b.posts(ctx, r)
	b.index(ctx, r)
	r.End = time.Now()
	log15.Debug("Build.Finish", "duration", r.End.Sub(r.Begin), "error", r.Error)
	if r.Error != nil {
		log15.Error("Build.Error", "error", r.Error.Error())
	}
	b.isBuilding = false
	b.report = r
}

func (b *Builder) IsBuilding() bool {
	return b.isBuilding
}

func (b *Builder) Report() *Report {
	return b.report
}
