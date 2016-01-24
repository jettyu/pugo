package builder

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/go-xiaohei/pugo/app/model"
)

// AssembleSource assemble some extra data in Source,
// such as page nodes, i18n status.
// it need be used after posts and pages are loaded
func AssembleSource(ctx *Context) {
	if ctx.Source == nil || ctx.Theme == nil {
		ctx.Err = fmt.Errorf("need sources data and theme to assemble")
		return
	}

	ctx.Source.Nav.FixURL(ctx.Source.Meta.Path)
	ctx.Source.Tags = make(map[string]*model.Tag)
	ctx.Source.tagPosts = make(map[string][]*model.Post)
	ctx.Tree = model.NewTree()

	r, hr := newReplacer(ctx.Source.Meta.Path), newReplacerInHTML(ctx.Source.Meta.Path)
	ctx.Source.Meta.Cover = r.Replace(ctx.Source.Meta.Cover)
	for _, a := range ctx.Source.Authors {
		a.Avatar = r.Replace(a.Avatar)
	}

	for _, p := range ctx.Source.Posts {
		if ctx.Source.Meta.Path != "" && ctx.Source.Meta.Path != "/" {
			p.FixURL(ctx.Source.Meta.Path)
		}
		p.FixPlaceholder(r, hr)
		p.Author = ctx.Source.Authors[p.AuthorName]
		for _, t := range p.Tags {
			ctx.Source.Tags[t.Name] = t
			ctx.Source.tagPosts[t.Name] = append(ctx.Source.tagPosts[t.Name], p)
		}
	}
	for _, p := range ctx.Source.Pages {
		if ctx.Source.Meta.Path != "" && ctx.Source.Meta.Path != "/" {
			p.FixURL(ctx.Source.Meta.Path)
		}
		p.FixPlaceholder(hr)
		p.Author = ctx.Source.Authors[p.AuthorName]
	}

	for _, posts := range ctx.Source.tagPosts {
		sort.Sort(model.Posts(posts))
	}

	ctx.Source.Archive = model.NewArchive(ctx.Source.Posts)

	if ctx.Err = ctx.Theme.Load(); ctx.Err != nil {
		return
	}
}

func newReplacer(static string) *strings.Replacer {
	return strings.NewReplacer(
		"@media", path.Join(static, "media"),
	)
}

func newReplacerInHTML(static string) *strings.Replacer {
	media := path.Join(static, "media")
	return strings.NewReplacer(
		`src="@media`, fmt.Sprintf(`src="%s`, media),
		`href="@media`, fmt.Sprintf(`src="%s`, media),
	)
}
