package codegen

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

type goPkg struct {
	path  string
	alias string
}

type goValue struct {
	*goPkg
	name string
}

func (o *goValue) String() string {
	if o.goPkg.alias == "" {
		return o.name
	}

	return o.goPkg.alias + "." + o.name
}

type byPkgPath []*goPkg

func (s byPkgPath) Len() int {
	return len(s)
}

func (s byPkgPath) Less(i int, j int) bool {
	return s[i].path < s[j].path
}

func (s byPkgPath) Swap(i int, j int) {
	s[i], s[j] = s[j], s[i]
}

type GoPkgMap interface {
	String() string

	Name(path string, name string) fmt.Stringer

	Reserve(name string)
	AssignAliases()
}

type goPkgMap struct {
	localPath string
	pkgs      map[string]*goPkg
	reserved  map[string]struct{}
}

func newGoPkgMap(localPath string) GoPkgMap {
	m := &goPkgMap{
		localPath: localPath,
		pkgs:      make(map[string]*goPkg),
		reserved:  make(map[string]struct{}),
	}

	m.pkgs[localPath] = &goPkg{}

	return m
}

func (m *goPkgMap) String() string {
	m.AssignAliases()

	golang := make([]*goPkg, 0)
	thirdParty := make([]*goPkg, 0)
	dropbox := make([]*goPkg, 0)

	for _, pkg := range m.pkgs {
		if pkg.alias == "" { // local package
			continue
		}

		root := strings.Split(pkg.path, "/")[0]
		if root == "dropbox" {
			dropbox = append(dropbox, pkg)
		} else if root == "godropbox" || len(strings.Split(root, ".")) > 1 {
			thirdParty = append(thirdParty, pkg)
		} else {
			golang = append(golang, pkg)
		}
	}

	if len(golang) == 0 && len(thirdParty) == 0 && len(dropbox) == 0 {
		return ""
	}

	sort.Sort(byPkgPath(golang))
	sort.Sort(byPkgPath(thirdParty))
	sort.Sort(byPkgPath(dropbox))

	hdr := NewLineWriter("\t")
	l := hdr.Line

	l("import (")
	hdr.PushIndent()

	for _, pkg := range golang {
		l("%s \"%s\"", pkg.alias, pkg.path)
	}

	if len(thirdParty) > 0 {
		if len(golang) > 0 {
			l("")
		}

		for _, pkg := range thirdParty {
			l("%s \"%s\"", pkg.alias, pkg.path)
		}
	}

	if len(dropbox) > 0 {
		if len(golang) > 0 || len(thirdParty) > 0 {
			l("")
		}

		for _, pkg := range dropbox {
			l("%s \"%s\"", pkg.alias, pkg.path)
		}
	}

	hdr.PopIndent()
	l(")")
	l("")

	return hdr.String()
}

func (m *goPkgMap) Reserve(name string) {
	m.reserved[name] = struct{}{}
}

func (m *goPkgMap) Name(path string, name string) fmt.Stringer {
	p, ok := m.pkgs[path]
	if !ok {
		p = &goPkg{
			path: path,
		}
		m.pkgs[path] = p
	}

	return &goValue{
		goPkg: p,
		name:  name,
	}
}

func (m *goPkgMap) AssignAliases() {
	_, localName := path.Split(m.localPath)

	paths := []string{}
	for pkgPath := range m.pkgs {
		paths = append(paths, pkgPath)
	}
	sort.Strings(paths)

	aliases := make(map[string]struct{})
	for name := range m.reserved {
		aliases[name] = struct{}{}
	}

	for _, pkgPath := range paths {
		if pkgPath == m.localPath {
			continue
		}

		_, name := path.Split(pkgPath)

		count := 0
		if name == localName { // same pkg name as local pkg
			count = 1
		}

		for {
			alias := name
			if count > 0 {
				alias = fmt.Sprintf("%s%d", name, count)
			}

			_, ok := aliases[alias]
			if !ok {
				m.pkgs[pkgPath].alias = alias
				aliases[alias] = struct{}{}
				break
			}

			count++
		}
	}
}

type GoWriter struct {
	codegenToolName string

	pkgName string

	GoPkgMap

	LineWriter
}

func NewGoWriter(codegenToolName string, pkg string, pkgPath string) *GoWriter {
	return &GoWriter{
		codegenToolName: codegenToolName,
		pkgName:         pkg,
		GoPkgMap:        newGoPkgMap(pkg),
		LineWriter:      NewLineWriter("\t"),
	}
}

func (w *GoWriter) String() string {
	hdr := fmt.Sprintf(
		"// Auto-generated by %s.  Do not edit!\n\npackage %s\n\n",
		w.codegenToolName,
		w.pkgName)

	return hdr + w.GoPkgMap.String() + w.LineWriter.String()
}
